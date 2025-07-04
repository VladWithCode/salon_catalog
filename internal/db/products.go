package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Product struct {
	ID          string            `db:"id" json:"id"`
	Name        string            `db:"name" json:"name"`
	Slug        string            `db:"slug" json:"slug"`
	Description string            `db:"description" json:"description"`
	MainImg     string            `db:"main_img" json:"mainImg"`
	Gallery     []string          `db:"gallery" json:"gallery"`
	Price       int               `db:"price" json:"price"`
	Features    map[string]string `db:"features" json:"features"`
	Category    string            `db:"category" json:"category"`
	CategoryID  string            `db:"category_id" json:"categoryId"`
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

	mainImg := sql.NullString{
		String: product.MainImg,
		Valid:  product.MainImg != "",
	}
	_, err = conn.Exec(
		ctx,
		`INSERT INTO products (id, name, slug, description, price, main_img, category, features) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id.String(),
		product.Name,
		product.Slug,
		product.Description,
		product.Price,
		mainImg,
		product.Category,
		product.Features,
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
	var mainImg sql.NullString
	var gallery pgtype.Array[*string]
	err = conn.QueryRow(
		ctx,
		`SELECT 
			prod.id, prod.name, prod.slug, prod.description,
			prod.price, prod.features,
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
		GROUP BY prod.id, prod.name, prod.slug, prod.description, prod.price, prod.features, main.filename, ctg.name, ctg.id`,
		slug,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Slug,
		&product.Description,
		&product.Price,
		&product.Features,
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
	err = conn.QueryRow(
		ctx,
		`SELECT 
			prod.id, prod.name, prod.slug, prod.description, prod.price,
			prod.features, 
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
		GROUP BY prod.id, prod.name, prod.slug, prod.description, prod.price, prod.features, main.filename, ctg.name, ctg.id`,
		id,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Slug,
		&product.Description,
		&product.Price,
		&product.Features,
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
			prod.id, prod.name, prod.slug, prod.description, prod.price,
			prod.features, 
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
		err = rows.Scan(
			&product.ID,
			&product.Name,
			&product.Slug,
			&product.Description,
			&product.Price,
			&product.Features,
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

	_, err = conn.Exec(
		ctx,
		`UPDATE products SET
			name = $1, slug = $2, description = $3, price = $4, features = $5, category = $6, main_img = $7
		WHERE id = $8`,
		product.Name,
		product.Slug,
		product.Description,
		product.Price,
		product.Features,
		product.Category,
		mainImg,
		product.ID,
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
