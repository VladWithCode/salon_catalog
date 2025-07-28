// Package db provides a connection pool to the database
// as well as functions to interact with the database
// per database tables and views.
package db

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoConnStr = errors.New("required env var DATABASE_URL is not set")
	ErrUUIDFail  = errors.New("failed to generate new uuid")
)

var dbPool *pgxpool.Pool

func Connect() (*pgxpool.Pool, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, ErrNoConnStr
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	dbPool = pool

	return pool, nil
}

func GetConn() (*pgxpool.Conn, error) {
	return dbPool.Acquire(context.Background())
}

type PaginationData struct {
	CurrentPage  int   `json:"current_page"`
	TotalPages   int   `json:"total_pages"`
	TotalItems   int   `json:"total_items"`
	ItemsPerPage int   `json:"items_per_page"`
	StartItem    int   `json:"start_item"`
	EndItem      int   `json:"end_item"`
	HasPrevious  bool  `json:"has_previous"`
	HasNext      bool  `json:"has_next"`
	PreviousPage int   `json:"previous_page"`
	NextPage     int   `json:"next_page"`
	Pages        []int `json:"pages"`
}

// buildOrderByClause constructs the ORDER BY clause based on sort parameter
func buildOrderByClause(sort string) string {
	switch strings.ToLower(sort) {
	case "name_asc", "name":
		return "ORDER BY name ASC"
	case "name_desc":
		return "ORDER BY name DESC"
	case "price_asc":
		return "ORDER BY price ASC"
	case "price_desc":
		return "ORDER BY price DESC"
	case "newest":
		return "ORDER BY id DESC" // Assuming newer products have newer IDs
	case "oldest":
		return "ORDER BY id ASC"
	case "category":
		return "ORDER BY category_name ASC, name ASC"
	default:
		return "ORDER BY name ASC" // Default sort
	}
}
