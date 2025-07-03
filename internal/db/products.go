package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	Id          string            `db:"id" json:"id"`
	Name        string            `db:"name" json:"name"`
	Slug        string            `db:"slug" json:"slug"`
	Description string            `db:"description" json:"description"`
	MainImg     string            `db:"main_img" json:"mainImg"`
	Gallery     []string          `db:"gallery" json:"gallery"`
	Price       int               `db:"price" json:"price"`
	Features    map[string]string `db:"features" json:"features"`
}

func CreateProduct(product *Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	id, err := uuid.NewV7()
	if err != nil {
		return ErrUUIDFail
	}

	_, err = conn.Exec(
		ctx,
		`INSERT INTO products (id, name, slug, description) VALUES ($1, $2, $3, $4)`,
		id.String(),
		product.Name,
		product.Slug,
		product.Description,
	)
	if err != nil {
		return err
	}

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
	err = conn.QueryRow(
		ctx,
		`SELECT id, name, slug, description FROM products WHERE slug = $1`,
		slug,
	).Scan(
		&product.Id,
		&product.Name,
		&product.Slug,
		&product.Description,
	)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func FindProductById(id string) (*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var product Product
	err = conn.QueryRow(
		ctx,
		`SELECT id, name, slug, description, FROM products WHERE id = $1`,
		id,
	).Scan(
		&product.Id,
		&product.Name,
		&product.Slug,
		&product.Description,
	)
	if err != nil {
		return nil, err
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
		`SELECT id, name, slug, description FROM products`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*Product
	for rows.Next() {
		var product Product
		err = rows.Scan(
			&product.Id,
			&product.Name,
			&product.Slug,
			&product.Description,
		)
		if err != nil {
			return nil, err
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

	_, err = conn.Exec(
		ctx,
		`UPDATE products SET name = $1, slug = $2, description = $3, price = $6 WHERE id = $7`,
		product.Name,
		product.Slug,
		product.Description,
		product.Price,
		product.Id,
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
