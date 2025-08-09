package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

// FindRelatedProducts finds products related to a given product ID
// It uses the pre-calculated similarity scores from the product_similarities materialized view
// for optimal performance. Falls back to category-based recommendations if needed.
func FindRelatedProducts(productID string, limit int) ([]*CatalogProd, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Set default limit
	if limit < 1 || limit > 20 {
		limit = 8
	}

	// First, resolve the product ID if a slug was provided
	var resolvedID string
	if _, err = uuid.Parse(productID); err == nil {
		err = conn.QueryRow(ctx, `
			SELECT id FROM products 
			WHERE id = $1
		`, productID).Scan(&resolvedID)
	} else {
		err = conn.QueryRow(ctx, `
			SELECT id FROM products 
			WHERE slug = $1
		`, productID).Scan(&resolvedID)
	}

	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	// Query using the materialized view for pre-calculated similarities
	query := `
		SELECT 
			p.id,
			p.name,
			p.description,
			p.long_description,
			p.category as category_id,
			c.name as category_name,
			COALESCE(i.filename, '') as image_url,
			p.available,
			p.slug,
			COALESCE(
				(
					SELECT json_agg(img.filename ORDER BY img.filename)
					FROM images_products ip
					JOIN images img ON ip.image_id = img.id
					WHERE ip.product_id = p.id
				),
				'[]'::json
			) as images,
			ps.similarity_score
		FROM product_similarities ps
		JOIN products p ON ps.related_id = p.id
		LEFT JOIN categories c ON p.category = c.id
		LEFT JOIN images i ON p.main_img = i.id
		WHERE ps.product_id = $1
			AND p.available = true
		ORDER BY ps.similarity_score DESC, p.name
		LIMIT $2
	`

	rows, err := conn.Query(ctx, query, resolvedID, limit)
	if err != nil {
		log.Printf("main strat failed: %v\n", err)
		// If the materialized view doesn't exist or has issues, fall back to direct calculation
		return findRelatedProductsFallback(ctx, conn, resolvedID, limit)
	}
	defer rows.Close()

	var products []*CatalogProd
	for rows.Next() {
		var product CatalogProd
		var imagesJSON []byte
		var similarityScore float64

		err = rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.LongDescription,
			&product.CategoryID,
			&product.CategoryName,
			&product.ImageURL,
			&product.Available,
			&product.Slug,
			&imagesJSON,
			&similarityScore, // We can use this for debugging or display if needed
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan related product: %w", err)
		}

		// Unmarshal JSON fields
		if err = json.Unmarshal(imagesJSON, &product.Images); err != nil {
			return nil, fmt.Errorf("failed to unmarshal images: %w", err)
		}

		products = append(products, &product)
	}

	// If we don't have enough related products, fill with products from the same category
	if len(products) < limit {
		remainingLimit := limit - len(products)

		// Get the current product's category
		var currentCategoryID string
		err = conn.QueryRow(ctx, `
			SELECT category FROM products WHERE id = $1
		`, resolvedID).Scan(&currentCategoryID)

		if err == nil && currentCategoryID != "" {
			// Get product IDs we already have
			existingIDs := make([]string, len(products)+1)
			existingIDs[0] = resolvedID // Don't include the current product
			for i, p := range products {
				existingIDs[i+1] = p.ID
			}

			// Query for additional products from the same category
			fallbackQuery := `
				SELECT 
					p.id,
					p.name,
					p.description,
					p.long_description,
					p.category as category_id,
					c.name as category_name,
					COALESCE(i.filename, '') as image_url,
					p.available,
					p.slug,
					COALESCE(
						(
							SELECT json_agg(img.filename ORDER BY img.filename)
							FROM images_products ip
							JOIN images img ON ip.image_id = img.id
							WHERE ip.product_id = p.id
						),
						'[]'::json
					) as images
				FROM products p
				LEFT JOIN categories c ON p.category = c.id
				LEFT JOIN images i ON p.main_img = i.id
				WHERE p.category = $1
					AND p.available = true
					AND p.id != ALL($2)
				ORDER BY RANDOM() -- Random for variety
				LIMIT $3
			`

			fallbackRows, err := conn.Query(ctx, fallbackQuery, currentCategoryID, existingIDs, remainingLimit)
			if err == nil {
				defer fallbackRows.Close()

				for fallbackRows.Next() {
					var product CatalogProd
					var imagesJSON []byte

					err = fallbackRows.Scan(
						&product.ID,
						&product.Name,
						&product.Description,
						&product.LongDescription,
						&product.CategoryID,
						&product.CategoryName,
						&product.ImageURL,
						&product.Available,
						&product.Slug,
						&imagesJSON,
					)
					if err == nil {
						if err = json.Unmarshal(imagesJSON, &product.Images); err == nil {
							products = append(products, &product)
						}
					}
				}
			}
		}
	}

	return products, nil
}

// findRelatedProductsFallback is used when the materialized view is not available
// It calculates similarities on-the-fly (slower but always works)
func findRelatedProductsFallback(ctx context.Context, conn *pgxpool.Conn, productID string, limit int) ([]*CatalogProd, error) {
	query := `
		WITH current_product AS (
			SELECT id, category, search_vector
			FROM products 
			WHERE id = $1
		)
		SELECT 
			p.id,
			p.name,
			p.description,
			p.long_description,
			p.category as category_id,
			c.name as category_name,
			COALESCE(i.filename, '') as image_url,
			p.available,
			p.slug,
			COALESCE(
				(
					SELECT json_agg(img.filename ORDER BY img.filename)
					FROM images_products ip
					JOIN images img ON ip.image_id = img.id
					WHERE ip.product_id = p.id
				),
				'[]'::json
			) as images
		FROM products p
		CROSS JOIN current_product cp
		LEFT JOIN categories c ON p.category = c.id
		LEFT JOIN images i ON p.main_img = i.id
		WHERE p.id != cp.id
			AND p.available = true
			AND (
				p.category = cp.category  -- Same category
				OR ts_rank(p.search_vector, cp.search_vector) > 0.1  -- Or similar content
			)
		ORDER BY 
			CASE WHEN p.category = cp.category THEN 0 ELSE 1 END,  -- Prioritize same category
			ts_rank(p.search_vector, cp.search_vector) DESC,
			p.name
		LIMIT $2
	`

	rows, err := conn.Query(ctx, query, productID, limit)
	if err != nil {
		return nil, fmt.Errorf("fallback query failed: %w", err)
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
			&product.Slug,
			&imagesJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fallback product: %w", err)
		}

		if err = json.Unmarshal(imagesJSON, &product.Images); err != nil {
			return nil, fmt.Errorf("failed to unmarshal images: %w", err)
		}

		products = append(products, &product)
	}

	return products, nil
}

// RefreshProductSimilarities triggers a refresh of the materialized view
// Call this periodically (e.g., via a cron job) or after bulk product updates
func RefreshProductSimilarities() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Longer timeout for refresh
	defer cancel()

	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY product_similarities")
	if err != nil {
		return fmt.Errorf("failed to refresh product similarities: %w", err)
	}

	return nil
}

// Alternative: Simpler version focusing on category-based recommendations
func FindRelatedProductsSimple(productID string, limit int) ([]*CatalogProd, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	if limit < 1 || limit > 20 {
		limit = 8
	}

	// Simple query: get products from the same category
	query := `
		SELECT 
			p.id,
			p.name,
			p.description,
			p.long_description,
			p.category as category_id,
			c.name as category_name,
			COALESCE(i.filename, '') as image_url,
			p.available,
			p.slug,
			COALESCE(
				(
					SELECT json_agg(img.filename ORDER BY img.filename)
					FROM images_products ip
					JOIN images img ON ip.image_id = img.id
					WHERE ip.product_id = p.id
				),
				'[]'::json
			) as images
		FROM products p
		JOIN products current_p ON (current_p.id = $1 OR current_p.slug = $1)
		LEFT JOIN categories c ON p.category = c.id
		LEFT JOIN images i ON p.main_img = i.id
		WHERE p.category = current_p.category
			AND p.id != current_p.id
			AND p.available = true
		ORDER BY RANDOM()  -- Random selection for variety
		LIMIT $2
	`

	rows, err := conn.Query(ctx, query, productID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find related products: %w", err)
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
			&product.Slug,
			&imagesJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan related product: %w", err)
		}

		if err = json.Unmarshal(imagesJSON, &product.Images); err != nil {
			return nil, fmt.Errorf("failed to unmarshal images: %w", err)
		}

		products = append(products, &product)
	}

	return products, nil
}
