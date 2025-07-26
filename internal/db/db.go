// Package db provides a connection pool to the database
// as well as functions to interact with the database
// per database tables and views.
package db

import (
	"context"
	"errors"
	"os"

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
