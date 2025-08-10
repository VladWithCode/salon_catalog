package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const DefaultCartCookieDuration = time.Hour * 24 * 30 // 30 days

var (
	ErrCartIDInvalidMissing       = errors.New("cart id is missing or invalid")
	ErrCartNotFound               = errors.New("cart not found")
	ErrCartSetFieldInvalid        = errors.New("invalid cart field")
	ErrCartSaveFailedData         = errors.New("failed to save cart data")
	ErrCartSaveFailedItems        = errors.New("failed to save cart items")
	ErrCartSaveFailedRemovedItems = errors.New("failed to save removed cart items")
)

type Cart struct {
	ID            string      `json:"id"`
	CustomerName  string      `json:"customer_name"`
	CustomerEmail string      `json:"customer_email"`
	CustomerPhone string      `json:"customer_phone"`
	Items         []*CartItem `json:"items"`
	CreatedAt     time.Time   `json:"created_at"`
	IsSubmitted   bool        `json:"is_submitted"`

	updatedFields map[string]bool `json:"-"`
	removedItems  []string        `json:"-"`
	isNew         bool            `json:"-"`
}

type CartItem struct {
	ProductID string    `json:"product_id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	ImageURL  string    `json:"image_url"`
	Quantity  int       `json:"quantity"`
	MaxQty    int       `json:"max_quantity"`
	Source    string    `json:"source"`               // "wizard" or "catalog"
	StepIndex int       `json:"step_index,omitempty"` // For wizard items
	CreatedAt time.Time `json:"created_at"`           // AddedAt
	UpdatedAt time.Time `json:"updated_at"`
}

type CartItemSource string

const (
	CartItemSourceWizard  CartItemSource = "asistente"
	CartItemSourceCatalog CartItemSource = "catálogo"
)

func NewCart(id ...string) *Cart {
	cartID := id[0]
	if cartID == "" {
		cartID = uuid.Must(uuid.NewV7()).String()
	}

	return &Cart{
		ID:            cartID,
		CustomerName:  "",
		CustomerEmail: "",
		CustomerPhone: "",
		Items:         make([]*CartItem, 0),
		CreatedAt:     time.Now(),

		updatedFields: make(map[string]bool),
		removedItems:  make([]string, 0),
		isNew:         true,
	}
}

func (c *Cart) GetField(key string) any {
	switch key {
	case "CustomerName", "customer_name":
		return c.CustomerName
	case "CustomerEmail", "customer_email":
		return c.CustomerEmail
	case "CustomerPhone", "customer_phone":
		return c.CustomerPhone
	case "Items", "items":
		return c.Items
	case "CreatedAt", "created_at":
		return c.CreatedAt
	case "IsSubmitted", "is_submitted":
		return c.IsSubmitted
	default:
		return nil
	}
}

func (c *Cart) SetField(key string, value any) error {
	switch key {
	case "CustomerName", "customer_name":
		c.CustomerName = value.(string)
	case "CustomerEmail", "customer_email":
		c.CustomerEmail = value.(string)
	case "CustomerPhone", "customer_phone":
		c.CustomerPhone = value.(string)
	case "Items", "items":
		items, valid := value.([]*CartItem)
		if valid {
			c.Items = items
		}
	case "CreatedAt", "created_at":
		c.CreatedAt = value.(time.Time)
	case "IsSubmitted", "is_submitted":
		c.IsSubmitted = value.(bool)
	default:
		return ErrCartSetFieldInvalid
	}

	return nil
}

// SetFrom receives a map of cart fields and updates the cart with the values
// from the map. It also marks the changed fields as updated.
func (c *Cart) SetFrom(fieldsPtr *map[string]any) {
	cartFields := *fieldsPtr
	for k, v := range cartFields {
		err := c.SetField(k, v)
		if err == nil {
			c.updatedFields[k] = true
		}
	}
}

func (c *Cart) AddItem(item *CartItem) {
	c.updatedFields["items"] = true
	exists := false
	for _, i := range c.Items {
		if i.ProductID == item.ProductID {
			exists = true
			i.Quantity += item.Quantity
			break
		}
	}
	if !exists {
		c.Items = append(c.Items, item)
	}
}

func (c *Cart) UpdateItemQty(itemID string, quantity int) {
	c.updatedFields["items"] = true
	if quantity <= 0 {
		c.RemoveItem(itemID)
		return
	}

	for i, item := range c.Items {
		if itemID == item.ProductID {
			c.Items[i].Quantity = min(quantity, item.MaxQty)
			break
		}
	}
}

func (c *Cart) RemoveItem(itemID string) {
	c.updatedFields["items"] = true
	c.removedItems = append(c.removedItems, itemID)
	if len(c.Items) == 0 {
		return
	}

	for i, item := range c.Items {
		if item.ProductID == itemID {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			break
		}
	}
}

func (c *Cart) ClearCart() {
	for _, item := range c.Items {
		c.removedItems = append(c.removedItems, item.ProductID)
	}

	c.Items = make([]*CartItem, 0)
}

func (c *Cart) GetItemQuantity(itemID string) int {
	for _, item := range c.Items {
		if item.ProductID == itemID {
			return item.Quantity
		}
	}
	return 0
}

func (c *Cart) GetItemMaxQty(itemID string) int {
	for _, item := range c.Items {
		if item.ProductID == itemID {
			return item.MaxQty
		}
	}
	return 0
}

func (c *Cart) Save(ctx context.Context) error {
	if c.ID == "" {
		return ErrCartIDInvalidMissing
	}
	conn, err := GetConnWithContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	// Insert/update cart data only if it's a new cart or there are updated fields
	// skip for saves with only item updates
	if (c.isNew || len(c.updatedFields) > 0) && !(len(c.updatedFields) == 1 && c.updatedFields["items"]) {
		baseQuery := `INSERT INTO carts`
		args := pgx.NamedArgs{"id": c.ID}

		columnStr := "(id"
		argStr := "(@id"
		updtStr := "ON CONFLICT (id) DO UPDATE SET"
		updtFields := make([]string, 0)
		for fld, updated := range c.updatedFields {
			if fld == "items" {
				continue
			}
			if updated {
				columnStr += ", " + fld
				argStr += ", @" + fld
				updtFields = append(updtFields, fmt.Sprintf("%s = @%s", fld, fld))
				args[fld] = c.GetField(fld)
			}
		}
		columnStr += ")"
		argStr += ")"
		updtStr += " " + strings.Join(updtFields, ", ")
		baseQuery = fmt.Sprintf("%s %s VALUES %s", baseQuery, columnStr, argStr)
		if len(updtFields) > 0 {
			baseQuery += " " + updtStr
		}

		_, err = tx.Exec(ctx, baseQuery, args)
	}

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to save cart: %w", err)
	}
	c.isNew = false

	if c.updatedFields["items"] {
		for _, item := range c.Items {
			args := pgx.NamedArgs{
				"cart_id":    c.ID,
				"product_id": item.ProductID,
				"quantity":   item.Quantity,
				"source":     item.Source,
				"created_at": item.CreatedAt,
			}
			_, err = tx.Exec(
				ctx,
				`INSERT INTO cart_items (cart_id, product_id, quantity, source, created_at, updated_at)
				VALUES (@cart_id, @product_id, @quantity, @source, @created_at, NOW())
				ON CONFLICT (cart_id, product_id) DO UPDATE SET
					quantity = @quantity,
					source = @source,
					updated_at = NOW()
			`,
				args,
			)
			if err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("failed to add items to cart: %w", err)
			}
		}
	}

	if len(c.removedItems) > 0 {
		_, err = tx.Exec(
			ctx,
			`DELETE FROM cart_items WHERE cart_id = $1 AND product_id = ANY($2)`,
			c.ID,
			c.removedItems,
		)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to remove items from cart: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// LoadItems loads all items for this cart from the database
func (c *Cart) LoadItems(ctx context.Context) error {
	conn, err := GetConnWithContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	rows, err := conn.Query(ctx, `
		SELECT ci.product_id, ci.quantity, ci.source, ci.created_at, ci.updated_at,
		       cp.name, cp.category_name, cp.image_url, cp.quantity as max_quantity
		FROM cart_items ci
		JOIN catalog_products cp ON ci.product_id = cp.id
		WHERE ci.cart_id = $1
		ORDER BY ci.created_at
	`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	c.Items = make([]*CartItem, 0)
	for rows.Next() {
		item := &CartItem{}
		err = rows.Scan(
			&item.ProductID, &item.Quantity, &item.Source, &item.CreatedAt, &item.UpdatedAt,
			&item.Name, &item.Category, &item.ImageURL, &item.MaxQty,
		)
		if err != nil {
			return err
		}
		c.Items = append(c.Items, item)
	}

	return rows.Err()
}

// FindCartByID loads a cart from the database by ID, including all its items
func FindCartByID(ctx context.Context, cartID string) (*Cart, error) {
	if cartID == "" {
		return nil, ErrCartIDInvalidMissing
	}

	conn, err := GetConnWithContext(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Load cart data
	cart := &Cart{
		ID:            cartID,
		updatedFields: make(map[string]bool),
		removedItems:  make([]string, 0),
		isNew:         false,
	}

	row := conn.QueryRow(ctx, `
		SELECT customer_name, customer_email, customer_phone, created_at, is_submitted
		FROM carts WHERE id = $1
	`, cartID)

	customerName := sql.NullString{}
	customerEmail := sql.NullString{}
	customerPhone := sql.NullString{}

	err = row.Scan(&customerName, &customerEmail, &customerPhone, &cart.CreatedAt, &cart.IsSubmitted)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrCartNotFound
		}
		return nil, err
	}
	cart.CustomerName = customerName.String
	cart.CustomerEmail = customerEmail.String
	cart.CustomerPhone = customerPhone.String

	// Load cart items
	err = cart.LoadItems(ctx)
	if err != nil {
		return nil, err
	}

	return cart, nil
}

// GetOrCreateCart gets an existing cart or creates a new one if it doesn't exist
func GetOrCreateCart(ctx context.Context, cartID string) (*Cart, error) {
	cart, err := FindCartByID(ctx, cartID)
	if err == ErrCartNotFound {
		// Create new cart
		cart = NewCart(cartID)
		err = cart.Save(ctx)
		if err != nil {
			return nil, err
		}
		return cart, nil
	}
	return cart, err
}

// DeleteCart removes a cart and all its items from the database
func DeleteCart(ctx context.Context, cartID string) error {
	if cartID == "" {
		return ErrCartIDInvalidMissing
	}

	conn, err := GetConnWithContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	// Delete cart (items will be deleted automatically due to CASCADE)
	_, err = conn.Exec(ctx, `DELETE FROM carts WHERE id = $1`, cartID)
	return err
}

// GetCartIDFromRequest extracts the cart ID from the request cookie
func GetCartIDFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie("cart_id")
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}
