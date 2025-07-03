package db

import (
	"context"
	"time"
)

type Image struct {
	ID         string    `db:"id" json:"id"`
	Filename   string    `db:"filename" json:"filename"`
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
			`INSERT INTO images (id, filename, no_optimize, size, created_at)
				VALUES ($1, $2, $3, $4, $5)`,
			img.ID,
			img.Filename,
			img.NoOptimize,
			img.Size,
			img.CreatedAt,
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
		`INSERT INTO images (id, filename, no_optimize, size, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		img.ID,
		img.Filename,
		img.NoOptimize,
		img.Size,
		img.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
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
		`SELECT id, filename, no_optimize, size, created_at FROM images WHERE id = $1`,
		id,
	).Scan(
		&image.ID,
		&image.Filename,
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
		`SELECT id, filename, no_optimize, size, created_at FROM images`,
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
