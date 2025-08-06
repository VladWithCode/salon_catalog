package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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

type Specification struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func FindCatalogCategories() ([]*CatalogCtg, error) {
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
			id, name, product_count
		FROM catalog_categories
		ORDER BY name`,
	)
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

func FindCatalogProductByID(id string) (*CatalogProd, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var (
		product    CatalogProd
		imagesJSON []byte
	)
	err = conn.QueryRow(
		ctx,
		`SELECT 
			id, name, description, long_description, category_id, category, 
			image_url, available, images
		FROM catalog_products WHERE id = $1`,
		id,
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
	)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(imagesJSON, &product.Images); err != nil {
		return nil, fmt.Errorf("failed to unmarshal images: %w", err)
	}

	return &product, nil
}

func FindCatalogProducts(categoryID string, search string) ([]*CatalogProd, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Build query conditionally
	query := `SELECT 
		id, name, description, long_description, category_id, category_name, 
		image_url, available, images
		FROM catalog_products WHERE 1=1`

	var args []any
	var conditions []string
	argIndex := 1

	// Add category filter if provided
	if categoryID != "" {
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", argIndex))
		args = append(args, categoryID)
		argIndex++
	}

	// Add search filter if provided
	if search != "" {
		searchPattern := "%" + search + "%"
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
		args = append(args, searchPattern)
		argIndex++
	}

	// Append conditions to query
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ordering
	query += " ORDER BY name"

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
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
			&imagesJSON, // Scan JSON as bytes first
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		if err = json.Unmarshal(imagesJSON, &product.Images); err != nil {
			return nil, fmt.Errorf("failed to unmarshal images: %w", err)
		}

		products = append(products, &product)
	}

	// Check for iteration errors
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
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
