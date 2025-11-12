package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx stdlib driver
	"github.com/pressly/goose/v3"
)

// Connect creates a pgx pool using the provided URL.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	if url == "" {
		return nil, errors.New("database url cannot be empty")
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// Migrate applies goose migrations located at dir using the provided URL.
func Migrate(ctx context.Context, url, dir string) error {
	if url == "" {
		return errors.New("database url cannot be empty")
	}
	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	goose.SetBaseFS(nil)
	if err = goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.UpContext(ctx, sqlDB, dir)
}
