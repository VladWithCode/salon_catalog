package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrProductInsert = errors.New("failed to insert product")
	ErrGalleryInsert = errors.New("failed to insert gallery images")
)

type Product struct {
	ID              string   `db:"id" json:"id"`
	Name            string   `db:"name" json:"name"`
	Slug            string   `db:"slug" json:"slug"`
	Description     string   `db:"description" json:"description"`
	LongDescription string   `db:"long_description" json:"longDescription"`
	MainImg         string   `db:"main_img" json:"mainImg"`
	Gallery         []string `db:"gallery" json:"gallery"`
	Category        string   `db:"category" json:"category"`
	CategoryID      string   `db:"category_id" json:"categoryId"`
	Available       bool     `db:"available" json:"available"`
}

// SearchMode defines how search should behave
type SearchMode string

const (
	SearchModeFullText SearchMode = "fulltext" // Use full-text search
	SearchModeExact    SearchMode = "exact"    // Use exact matching
	SearchModeFuzzy    SearchMode = "fuzzy"    // Use LIKE matching (fallback)
)

type ProductFilterParams struct {
	Search     string     `json:"search"`
	SearchMode SearchMode `json:"search_mode"`
	Category   string     `json:"category"`
	Sort       string     `json:"sort"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
}

type ProductFilterResult struct {
	Products    []*Product `json:"products"`
	Total       int        `json:"total"`
	Page        int        `json:"page"`
	Limit       int        `json:"limit"`
	TotalPages  int        `json:"total_pages"`
	HasNext     bool       `json:"has_next"`
	HasPrevious bool       `json:"has_previous"`
}

func CreateProduct(product *Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	id, err := uuid.NewV7()
	if err != nil {
		return ErrUUIDFail
	}
	product.ID = id.String()
	mainImg := uuid.NullUUID{}
	if product.MainImg != "" {
		id, err := uuid.Parse(product.MainImg)
		mainImg.UUID = id
		mainImg.Valid = err == nil
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO products (id, name, slug, description, long_description, main_img, category) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		product.ID,
		product.Name,
		product.Slug,
		product.Description,
		product.LongDescription,
		mainImg,
		product.CategoryID,
	)
	if err != nil {
		return errors.Join(ErrProductInsert, err)
	}

	for _, img := range product.Gallery {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO images_products (image_id, product_id) VALUES ($1, $2)`,
			img,
			product.ID,
		)
		if err != nil {
			return errors.Join(ErrGalleryInsert, err)
		}
	}

	tx.Commit(ctx)
	return nil
}

func FindProductBySlug(slug string) (*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var product Product
	var mainImg sql.NullString
	var gallery pgtype.Array[*string]
	var longDescription sql.NullString
	err = conn.QueryRow(
		ctx,
		`SELECT 
			prod.id, prod.name, prod.slug, prod.description, prod.long_description,
			ctg.name AS category,
			ctg.id AS category_id,
			main.filename AS main_img,
			ARRAY_AGG(img.filename) AS gallery
		FROM products prod 
			LEFT JOIN images_products img_prod ON prod.id = img_prod.product_id
			LEFT JOIN images img ON img_prod.image_id = img.id
			LEFT JOIN images main ON main.id = prod.main_img
			LEFT JOIN categories ctg ON ctg.id = prod.category
		WHERE prod.slug = $1
		GROUP BY prod.id, prod.name, prod.slug, prod.description, prod.long_description, main.filename, ctg.name, ctg.id`,
		slug,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Slug,
		&product.Description,
		&longDescription,
		&product.Category,
		&product.CategoryID,
		&mainImg,
		&gallery,
	)
	if err != nil {
		return nil, err
	}

	if mainImg.Valid {
		product.MainImg = mainImg.String
	}
	if gallery.Valid {
		for _, img := range gallery.Elements {
			if img != nil {
				product.Gallery = append(product.Gallery, *img)
			}
		}
	}
	if longDescription.Valid {
		product.LongDescription = longDescription.String
	}

	return &product, nil
}

func FindProductByID(id string) (*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var product Product
	var mainImg sql.NullString
	var gallery pgtype.Array[*string]
	var longDescription sql.NullString
	err = conn.QueryRow(
		ctx,
		`SELECT 
			prod.id, prod.name, prod.slug, prod.description, prod.long_description,
			ctg.name AS category,
			ctg.id AS category_id,
			main.filename AS main_img,
			ARRAY_AGG(img.filename) AS gallery
		FROM products prod
			LEFT JOIN images_products img_prod ON prod.id = img_prod.product_id
			LEFT JOIN images img ON img_prod.image_id = img.id
			LEFT JOIN images main ON main.id = prod.main_img
			LEFT JOIN categories ctg ON ctg.id = prod.category
		WHERE prod.id = $1
		GROUP BY prod.id, prod.name, prod.slug, prod.description, prod.long_description, main.filename, ctg.name, ctg.id`,
		id,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Slug,
		&product.Description,
		&longDescription,
		&product.Category,
		&product.CategoryID,
		&mainImg,
		&gallery,
	)
	if err != nil {
		return nil, err
	}
	if mainImg.Valid {
		product.MainImg = mainImg.String
	}
	if gallery.Valid {
		for _, img := range gallery.Elements {
			if img != nil {
				product.Gallery = append(product.Gallery, *img)
			}
		}
	}
	if longDescription.Valid {
		product.LongDescription = longDescription.String
	}

	return &product, nil
}

func FindAllProducts() ([]*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, err := conn.Query(
		ctx,
		`SELECT 
			prod.id, prod.name, prod.slug, prod.description, prod.long_description,
			ctg.name AS category,
			ctg.id AS category_id,
			img.filename AS main_img
		FROM products prod
			LEFT JOIN images img ON img.id = prod.main_img
			LEFT JOIN categories ctg ON ctg.id = prod.category`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*Product
	for rows.Next() {
		var product Product
		var mainImg sql.NullString
		var longDescription sql.NullString
		err = rows.Scan(
			&product.ID,
			&product.Name,
			&product.Slug,
			&product.Description,
			&longDescription,
			&product.Category,
			&product.CategoryID,
			&mainImg,
		)
		if err != nil {
			return nil, err
		}
		if mainImg.Valid {
			product.MainImg = mainImg.String
		}
		if longDescription.Valid {
			product.LongDescription = longDescription.String
		}
		products = append(products, &product)
	}

	return products, nil
}

func UpdateProduct(product *Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	mainImg := sql.NullString{
		String: product.MainImg,
		Valid:  product.MainImg != "",
	}
	args := pgx.NamedArgs{
		"id":               product.ID,
		"name":             product.Name,
		"slug":             product.Slug,
		"description":      product.Description,
		"long_description": product.LongDescription,
		"category":         product.CategoryID,
		"main_img":         mainImg,
	}
	_, err = conn.Exec(
		ctx,
		`UPDATE products SET
			name = @name, slug = @slug, description = @description,
			long_description = @long_description, category = @category,
			main_img = @main_img
		WHERE id = @id`,
		args,
	)
	if err != nil {
		return err
	}

	return nil
}

func DeleteProduct(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`DELETE FROM products WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	return nil
}

func FilterProducts(filters ProductFilterParams) (*ProductFilterResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Set defaults
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 || filters.Limit > 100 {
		filters.Limit = 20
	}
	if filters.SearchMode == "" {
		filters.SearchMode = SearchModeFullText
	}

	// Build query conditions and named arguments
	conditions, namedArgs := buildQueryConditions(filters)

	// Base query with explicit column selection
	baseQuery := `
		FROM products prod
		LEFT JOIN categories ctg ON prod.category = ctg.id
		LEFT JOIN images_products imgs_prod ON imgs_prod.product_id = prod.id
		LEFT JOIN images imgs ON imgs_prod.image_id = imgs.id
		LEFT JOIN images img ON prod.main_img = img.id
		`
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err = conn.QueryRow(ctx, countQuery, namedArgs).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Calculate pagination
	offset := (filters.Page - 1) * filters.Limit
	totalPages := int(math.Ceil(float64(total) / float64(filters.Limit)))

	// Add pagination to named args
	namedArgs["limit"] = filters.Limit
	namedArgs["offset"] = offset

	// Build final query with sorting and pagination
	orderBy := buildProductsOrderByClause(filters)
	selectQuery := fmt.Sprintf(`
		SELECT 
			prod.id, prod.name, prod.description, prod.long_description, ctg.id as category_id, ctg.name as category,
			img.filename as main_img, prod.available, 
			COALESCE(ARRAY_AGG(imgs.filename) FILTER (WHERE imgs.filename IS NOT NULL), '{}') as images,
			%s
		%s GROUP BY prod.id, prod.name, prod.description, prod.long_description,
		ctg.id, ctg.name, img.filename, prod.available %s 
		LIMIT @limit OFFSET @offset`,
		buildSearchRankSelect(filters), baseQuery, orderBy)

	// Execute query
	rows, err := conn.Query(ctx, selectQuery, namedArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Scan results
	products, err := scanProducts(rows, filters.SearchMode == SearchModeFullText)
	if err != nil {
		return nil, err
	}

	// Build result
	result := &ProductFilterResult{
		Products:    products,
		Total:       total,
		Page:        filters.Page,
		Limit:       filters.Limit,
		TotalPages:  totalPages,
		HasNext:     filters.Page < totalPages,
		HasPrevious: filters.Page > 1,
	}

	return result, nil
}

// buildQueryConditions creates WHERE conditions and named arguments
func buildQueryConditions(filters ProductFilterParams) ([]string, pgx.NamedArgs) {
	var conditions []string
	namedArgs := make(pgx.NamedArgs)

	// Add search condition
	if filters.Search != "" {
		switch filters.SearchMode {
		case SearchModeFullText:
			// Full-text search with ranking
			conditions = append(conditions, "prod.search_vector @@ plainto_tsquery('spanish', @search_query)")
			namedArgs["search_query"] = filters.Search

		case SearchModeExact:
			// Exact match search
			conditions = append(conditions, "(name ILIKE @exact_search OR description ILIKE @exact_search)")
			namedArgs["exact_search"] = filters.Search

		case SearchModeFuzzy:
			// Fuzzy search (LIKE with wildcards)
			conditions = append(conditions, "(name ILIKE @fuzzy_search OR description ILIKE @fuzzy_search OR category_name ILIKE @fuzzy_search)")
			namedArgs["fuzzy_search"] = "%" + filters.Search + "%"
		}
	}

	// Add category filter
	if filters.Category != "" {
		conditions = append(conditions, "category_id = @category_id")
		namedArgs["category_id"] = filters.Category
	}

	return conditions, namedArgs
}

// buildSearchRankSelect adds search ranking column when using full-text search
func buildSearchRankSelect(filters ProductFilterParams) string {
	if filters.Search != "" && filters.SearchMode == SearchModeFullText {
		return "ts_rank(prod.search_vector, plainto_tsquery('spanish', @search_query)) as search_rank"
	}
	return "0 as search_rank"
}

// buildProductsOrderByClause constructs the ORDER BY clause
func buildProductsOrderByClause(filters ProductFilterParams) string {
	// If using full-text search with a query, prioritize search ranking
	if filters.Search != "" && filters.SearchMode == SearchModeFullText {
		switch strings.ToLower(filters.Sort) {
		case "relevance", "":
			return "ORDER BY search_rank DESC, name ASC"
		case "name_asc", "name":
			return "ORDER BY name ASC"
		case "name_desc":
			return "ORDER BY name DESC"
		case "price_asc":
			return "ORDER BY price ASC, search_rank DESC"
		case "price_desc":
			return "ORDER BY price DESC, search_rank DESC"
		default:
			return "ORDER BY search_rank DESC, name ASC"
		}
	}

	// Regular sorting without search ranking
	switch strings.ToLower(filters.Sort) {
	case "name_asc", "name", "":
		return "ORDER BY name ASC"
	case "name_desc":
		return "ORDER BY name DESC"
	case "price_asc":
		return "ORDER BY price ASC"
	case "price_desc":
		return "ORDER BY price DESC"
	case "newest":
		return "ORDER BY id DESC"
	case "oldest":
		return "ORDER BY id ASC"
	case "category":
		return "ORDER BY category_name ASC, name ASC"
	default:
		return "ORDER BY name ASC"
	}
}

// scanProducts scans the query results into CatalogProd structs
func scanProducts(rows pgx.Rows, includeRank bool) ([]*Product, error) {
	var products []*Product

	for rows.Next() {
		var product Product
		var images []string
		var searchRank float32
		var mainImg sql.NullString
		var longDescription sql.NullString

		if includeRank {
			err := rows.Scan(
				&product.ID,
				&product.Name,
				&product.Description,
				&longDescription,
				&product.CategoryID,
				&product.Category,
				&mainImg,
				&product.Available,
				&images,
				&searchRank,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan product with rank: %w", err)
			}
		} else {
			err := rows.Scan(
				&product.ID,
				&product.Name,
				&product.Description,
				&product.CategoryID,
				&product.Category,
				&mainImg,
				&product.Available,
				&images,
				&searchRank, // Still need to scan the rank column (will be 0)
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan product: %w", err)
			}
		}

		if mainImg.Valid {
			product.MainImg = mainImg.String
		}
		if len(images) > 0 {
			product.Gallery = images
		}
		if longDescription.Valid {
			product.LongDescription = longDescription.String
		}

		products = append(products, &product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return products, nil
}
