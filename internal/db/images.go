package db

import (
	"context"
	"time"
)

type Image struct {
	ID           string `db:"id" json:"id"`
	PublicURL    string `db:"public_url" json:"publicUrl"`
	InternalPath string `db:"internal_path" json:"internalPath"`
	NoOptimize   bool   `db:"no_optimize" json:"noOptimize"`
	Size         int    `db:"size" json:"size"`
	Group        string `db:"group" json:"group"`
	Tags         string `db:"tags" json:"tags"`
	CreatedAt    string `db:"created_at" json:"createdAt"`
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
		`INSERT INTO images (id, public_url, internal_path, no_optimize, size, group, tags, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		img.Id,
		img.PublicUrl,
		img.InternalPath,
		img.NoOptimize,
		img.Size,
		img.Group,
		img.Tags,
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
		`SELECT id, public_url, internal_path, no_optimize, size, group, tags, created_at FROM images WHERE id = $1`,
		id,
	).Scan(
		&image.Id,
		&image.PublicUrl,
		&image.InternalPath,
		&image.NoOptimize,
		&image.Size,
		&image.Group,
		&image.Tags,
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
		`SELECT id, public_url, internal_path, no_optimize, size, group, tags, created_at FROM images`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []*Image
	for rows.Next() {
		var image Image
		err = rows.Scan(
			&image.Id,
			&image.PublicUrl,
			&image.InternalPath,
			&image.NoOptimize,
			&image.Size,
			&image.Group,
			&image.Tags,
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
		`UPDATE images SET public_url = $1, internal_path = $2, no_optimize = $3, size = $4, group = $5, tags = $6, created_at = $7 WHERE id = $8`,
		image.PublicUrl,
		image.InternalPath,
		image.NoOptimize,
		image.Size,
		image.Group,
		image.Tags,
		image.CreatedAt,
		image.Id,
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
