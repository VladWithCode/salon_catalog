package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID          string `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	Slug        string `db:"slug" json:"slug"`
	Description string `db:"description" json:"description"`
	HeaderImg   string `db:"header_img" json:"headerImg"`
	DisplayImg  string `db:"display_img" json:"displayImg"`
}

func CreateCategory(category *Category) error {
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

	headerImg := sql.NullString{
		String: category.HeaderImg,
		Valid:  category.HeaderImg != "",
	}
	displayImg := sql.NullString{
		String: category.DisplayImg,
		Valid:  category.DisplayImg != "",
	}

	_, err = conn.Exec(
		ctx,
		`INSERT INTO categories (id, name, slug, description, header_img, display_img) VALUES ($1, $2, $3, $4, $5, $6)`,
		id.String(),
		category.Name,
		category.Slug,
		category.Description,
		headerImg,
		displayImg,
	)
	if err != nil {
		return err
	}

	return nil
}

func FindCategoryBySlug(slug string) (*Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var (
		category   Category
		headerImg  sql.NullString
		displayImg sql.NullString
	)

	err = conn.QueryRow(
		ctx,
		`SELECT 
			ctg.id, ctg.name, ctg.slug, ctg.description,
			header.filename AS header_img,
			display.filename AS display_img
		FROM categories ctg
			JOIN images header ON header.id = ctg.header_img
			JOIN images display ON display.id = ctg.display_img
		WHERE slug = $1`,
		slug,
	).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.Description,
		&headerImg,
		&displayImg,
	)
	if err != nil {
		return nil, err
	}

	if headerImg.Valid {
		category.HeaderImg = headerImg.String
	}
	if displayImg.Valid {
		category.DisplayImg = displayImg.String
	}

	return &category, nil
}

func FindCategoryByID(id string) (*Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var (
		category   Category
		headerImg  sql.NullString
		displayImg sql.NullString
	)

	err = conn.QueryRow(
		ctx,
		`SELECT 
			ctg.id, ctg.name, ctg.slug, ctg.description,
			header.filename AS header_img,
			display.filename AS display_img
		FROM categories ctg
			LEFT JOIN images header ON header.id = ctg.header_img
			LEFT JOIN images display ON display.id = ctg.display_img
		WHERE ctg.id = $1`,
		id,
	).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.Description,
		&headerImg,
		&displayImg,
	)
	if err != nil {
		return nil, err
	}

	if headerImg.Valid {
		category.HeaderImg = headerImg.String
	}
	if displayImg.Valid {
		category.DisplayImg = displayImg.String
	}

	return &category, nil
}

func FindAllCategories() ([]*Category, error) {
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
			ctg.id, ctg.name, ctg.slug, ctg.description, 
			header.filename AS header_img,
			display.filename AS display_img
		FROM categories ctg
			LEFT JOIN images header ON header.id = ctg.header_img
			LEFT JOIN images display ON display.id = ctg.display_img
		ORDER BY ctg.name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*Category
	for rows.Next() {
		var (
			category   Category
			headerImg  sql.NullString
			displayImg sql.NullString
		)

		err = rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.Description,
			&headerImg,
			&displayImg,
		)
		if err != nil {
			return nil, err
		}

		if headerImg.Valid {
			category.HeaderImg = headerImg.String
		}
		if displayImg.Valid {
			category.DisplayImg = displayImg.String
		}

		categories = append(categories, &category)
	}

	return categories, nil
}

func UpdateCategory(category *Category) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	headerImg := sql.NullString{
		String: category.HeaderImg,
		Valid:  category.HeaderImg != "",
	}
	displayImg := sql.NullString{
		String: category.DisplayImg,
		Valid:  category.DisplayImg != "",
	}
	_, err = conn.Exec(
		ctx,
		`UPDATE categories SET
			name = $1, slug = $2, description = $3, header_img = $4, display_img = $5 
		WHERE id = $6`,
		category.Name,
		category.Slug,
		category.Description,
		headerImg,
		displayImg,
		category.ID,
	)
	if err != nil {
		return err
	}

	return nil
}

func DeleteCategory(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`DELETE FROM categories WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	return nil
}
