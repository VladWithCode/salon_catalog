package db

import (
	"context"
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
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	LongDescription string          `json:"long_description"`
	Category        string          `json:"category"`
	CategoryID      string          `json:"category_id"`
	ImageURL        string          `json:"image_url"`
	Images          []string        `json:"images"`
	Price           int             `json:"price"`
	PriceType       string          `json:"price_type"` // "por día", "por evento", etc.
	Available       bool            `json:"available"`
	Specifications  []Specification `json:"specifications"`
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
		specsJSON  []byte
	)
	err = conn.QueryRow(
		ctx,
		`SELECT 
			id, name, description, long_description, category_id, category_name, 
			image_url, price, price_type, available, images, specifications 
		FROM catalog_products WHERE id = $1`,
		id,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.LongDescription,
		&product.CategoryID,
		&product.Category,
		&product.ImageURL,
		&product.Price,
		&product.PriceType,
		&product.Available,
		&imagesJSON,
		&specsJSON,
	)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(imagesJSON, &product.Images); err != nil {
		return nil, fmt.Errorf("failed to unmarshal images: %w", err)
	}

	if err = json.Unmarshal(specsJSON, &product.Specifications); err != nil {
		return nil, fmt.Errorf("failed to unmarshal specifications: %w", err)
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
		image_url, price, price_type, available, images, specifications 
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
		var imagesJSON, specsJSON []byte

		err = rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.LongDescription,
			&product.CategoryID,
			&product.Category,
			&product.ImageURL,
			&product.Price,
			&product.PriceType,
			&product.Available,
			&imagesJSON, // Scan JSON as bytes first
			&specsJSON,  // Scan JSON as bytes first
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		if err = json.Unmarshal(imagesJSON, &product.Images); err != nil {
			return nil, fmt.Errorf("failed to unmarshal images: %w", err)
		}

		if err = json.Unmarshal(specsJSON, &product.Specifications); err != nil {
			return nil, fmt.Errorf("failed to unmarshal specifications: %w", err)
		}

		products = append(products, &product)
	}

	// Check for iteration errors
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}
