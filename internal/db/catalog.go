package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	DefaultCatalogPageSize = 16
)

type CatalogCtg struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProductCount int    `json:"product_count"`
}

type CatalogProd struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Slug            string   `json:"slug"`
	Description     string   `json:"description"`
	LongDescription string   `json:"long_description"`
	CategoryName    string   `json:"category"`
	CategoryID      string   `json:"category_id"`
	ImageURL        string   `json:"image_url"`
	Images          []string `json:"images"`
	Available       bool     `json:"available"`
}

func FindCatalogCategories(search string) ([]*CatalogCtg, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Build query conditionally
	query := `SELECT 
		id, name, product_count
	FROM catalog_categories WHERE 1=1`

	var args []any
	var conditions []string
	argIndex := 1

	// Add search filter if provided
	if search != "" {
		conditions = append(conditions, fmt.Sprintf("search_vector @@ plainto_tsquery('spanish', $%d)", argIndex))
		args = append(args, search)
		argIndex++
	}

	// Append conditions to query
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ordering - prioritize search ranking if search is provided
	if search != "" {
		query += " ORDER BY ts_rank(search_vector, plainto_tsquery('spanish', $1)) DESC, name ASC"
	} else {
		query += " ORDER BY name"
	}

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*CatalogCtg
	for rows.Next() {
		var category CatalogCtg
		err = rows.Scan(
			&category.ID,
			&category.Name,
			&category.ProductCount,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, &category)
	}

	return categories, nil
}

func FindCatalogProductDetail(id string) (*CatalogProd, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	baseQuery := `SELECT 
		id, name, description, long_description, category_id, category_name, 
		image_url, available, images, slug
	FROM catalog_products WHERE`
	args := pgx.NamedArgs{}

	if _, err := uuid.Parse(id); err == nil {
		baseQuery += " id = @id"
		args["id"] = id
	} else {
		baseQuery += " slug = @slug"
		args["slug"] = id
	}

	var (
		product    CatalogProd
		imagesJSON []byte
	)
	err = conn.QueryRow(
		ctx,
		baseQuery,
		args,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.LongDescription,
		&product.CategoryID,
		&product.CategoryName,
		&product.ImageURL,
		&product.Available,
		&imagesJSON,
		&product.Slug,
	)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(imagesJSON, &product.Images); err != nil {
		return nil, fmt.Errorf("failed to unmarshal images: %w", err)
	}

	return &product, nil
}

type CatalogProductFilterResult struct {
	Products    []*CatalogProd `json:"products"`
	Total       int            `json:"total"`
	Page        int            `json:"page"`
	Limit       int            `json:"limit"`
	TotalPages  int            `json:"total_pages"`
	HasNext     bool           `json:"has_next"`
	HasPrevious bool           `json:"has_previous"`
	HasError    bool           `json:"has_error"`
	Error       string         `json:"error"`
}

func FindCatalogProducts(categoryID string, search string, page int, limit int) (*CatalogProductFilterResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Set defaults
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = DefaultCatalogPageSize
	}

	// Build base query conditions
	var conditions []string
	args := pgx.NamedArgs{}

	// Add category filter if provided
	if categoryID != "" {
		if _, err := uuid.Parse(categoryID); err == nil {
			conditions = append(conditions, "category_id = @category_id")
			args["category_id"] = categoryID
		} else {
			conditions = append(conditions, "category_name = @category_name")
			args["category_name"] = categoryID
		}
	}

	// Add search filter if provided
	if search != "" {
		conditions = append(conditions, "search_vector @@ plainto_tsquery('spanish', @search)")
		args["search"] = search
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM catalog_products WHERE 1=1` + whereClause
	var total int
	err = conn.QueryRow(ctx, countQuery, args).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Calculate pagination
	offset := (page - 1) * limit
	totalPages := int((total + limit - 1) / limit) // Ceiling division

	// Add pagination parameters
	args["limit"] = limit
	args["offset"] = offset

	// Build main query with ordering and pagination
	query := `SELECT 
		id, name, description, long_description, category_id, category_name, 
		image_url, available, images, slug
		FROM catalog_products WHERE 1=1` + whereClause

	// Add ordering - prioritize search ranking if search is provided
	if search != "" {
		query += " ORDER BY ts_rank(search_vector, plainto_tsquery('spanish', @search)) DESC, name ASC"
	} else {
		query += " ORDER BY name"
	}

	query += " LIMIT @limit OFFSET @offset"

	rows, err := conn.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var products []*CatalogProd
	for rows.Next() {
		var product CatalogProd
		var imagesJSON []byte

		err = rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.LongDescription,
			&product.CategoryID,
			&product.CategoryName,
			&product.ImageURL,
			&product.Available,
			&imagesJSON,
			&product.Slug,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}

		// Unmarshal JSON fields
		if err = json.Unmarshal(imagesJSON, &product.Images); err != nil {
			return nil, fmt.Errorf("failed to unmarshal images: %w", err)
		}

		products = append(products, &product)
	}

	// Check for iteration errors
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Build result
	result := &CatalogProductFilterResult{
		Products:    products,
		Total:       total,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}

	return result, nil
}

func FindCatalogListings() (map[string][]*CatalogProd, error) {
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
			prod.id, prod.name, prod.description, prod.slug,
			ctg.name as category, pic.filename as main_img
		FROM (
			SELECT
				ROW_NUMBER() OVER (PARTITION BY p.category) as row_num,
				p.id, p.name, p.description, p.slug, p.category as category_id,
				p.main_img
			FROM products p
			ORDER BY p.category, p.name
		) as prod
		LEFT JOIN categories ctg ON prod.category_id = ctg.id
		LEFT JOIN images pic ON prod.main_img = pic.id
		WHERE prod.row_num <= 4
		ORDER BY category, prod.name
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	listings := make(map[string][]*CatalogProd)
	for rows.Next() {
		var product CatalogProd
		var imgUrl sql.NullString

		err = rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.Slug,
			&product.CategoryName,
			&imgUrl,
		)
		if err != nil {
			return nil, err
		}

		if imgUrl.Valid {
			product.ImageURL = imgUrl.String
		}
		if _, ok := listings[product.CategoryName]; !ok {
			listings[product.CategoryName] = []*CatalogProd{}
			listings[product.CategoryName] = append(listings[product.CategoryName], &product)
		} else {
			listings[product.CategoryName] = append(listings[product.CategoryName], &product)
		}
	}

	// Check for iteration errors
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return listings, nil
}
