package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PkgProduct struct {
	Name string `db:"name" json:"name"`
	Slug string `db:"slug" json:"slug"`
}

type EventPkg struct {
	Id               string     `db:"id" json:"id"`
	Name             string     `db:"name" json:"name"`
	Slug             string     `db:"slug" json:"slug"`
	Description      string     `db:"description" json:"description"`
	Price            int        `db:"price" json:"price"`
	IncludedProducts []*Product `db:"included_products" json:"includedProducts"`
	Savings          int        `db:"savings" json:"savings"`

	// TODO: IncludedServices
}

func CreatePackage(evtPkg *EventPkg) error {
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
		`INSERT INTO packages (id, name, slug, description, price) VALUES ($1, $2, $3, $4, $5)`,
		id.String(),
		evtPkg.Name,
		evtPkg.Slug,
		evtPkg.Description,
		evtPkg.Price,
	)
	if err != nil {
		return err
	}

	return nil
}
