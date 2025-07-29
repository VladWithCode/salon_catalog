package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrImageInsert                = errors.New("failed to insert image")
	ErrDeleteImageProductRelation = errors.New("failed to delete image product relation")
)

type Image struct {
	ID         string    `db:"id" json:"id"`
	Filename   string    `db:"filename" json:"filename"`
	Name       string    `db:"name" json:"name"`
	NoOptimize bool      `db:"no_optimize" json:"noOptimize"`
	Size       int       `db:"size" json:"size"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`
}

func CreateImages(imgs []*Image) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	for _, img := range imgs {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO images (id, filename, name, no_optimize, size)
				VALUES ($1, $2, $3, $4, $5)`,
			img.ID,
			img.Filename,
			img.Name,
			img.NoOptimize,
			img.Size,
		)

		if err != nil {
			return err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func CreateImage(img *Image) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`INSERT INTO images (id, filename, name, no_optimize, size) VALUES ($1, $2, $3, $4, $5)`,
		img.ID,
		img.Filename,
		img.Name,
		img.NoOptimize,
		img.Size,
	)
	if err != nil {
		return err
	}

	return nil
}

func LinkImagesToProduct(imgIDs []string, prodID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	for _, id := range imgIDs {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO images_products (image_id, product_id)
				VALUES ($1, $2)`,
			id,
			prodID,
		)

		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(
		ctx,
		`DELETE FROM images WHERE id NOT IN @ids::uuid[]`,
		pgx.NamedArgs{"ids": imgIDs},
	)
	if err != nil {
		return ErrDeleteImageProductRelation
	}

	return tx.Commit(ctx)
}

func UnlinkImagesFromProduct(imgIDs []string, prodID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	for _, id := range imgIDs {
		_, err = tx.Exec(
			ctx,
			`DELETE FROM images_products WHERE image_id = $1 AND product_id = $2`,
			id,
			prodID,
		)

		if err != nil {
			return ErrDeleteImageProductRelation
		}
	}

	return tx.Commit(ctx)
}

func FindImageByID(id string) (*Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var image Image
	err = conn.QueryRow(
		ctx,
		`SELECT id, filename, name, no_optimize, size, created_at FROM images WHERE id = $1`,
		id,
	).Scan(
		&image.ID,
		&image.Filename,
		&image.Name,
		&image.NoOptimize,
		&image.Size,
		&image.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &image, nil
}

func FindAllImages() ([]*Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, err := conn.Query(
		ctx,
		`SELECT id, filename, name, no_optimize, size, created_at FROM images`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []*Image
	for rows.Next() {
		var image Image
		err = rows.Scan(
			&image.ID,
			&image.Filename,
			&image.Name,
			&image.NoOptimize,
			&image.Size,
			&image.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		images = append(images, &image)
	}

	return images, nil
}

func UpdateImage(image *Image) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`UPDATE images SET filename = $1, no_optimize = $2, size = $3, created_at = $4 WHERE id = $5`,
		image.Filename,
		image.NoOptimize,
		image.Size,
		image.CreatedAt,
		image.ID,
	)

	if err != nil {
		return err
	}

	return nil
}

func DeleteImage(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`DELETE FROM images WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}

	return nil
}
