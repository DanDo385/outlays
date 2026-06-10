// Package store owns persistence: Postgres (system of record, append-only) and S3-compatible
// object storage for raw snapshots. Migrations are run with goose against the owner role; the
// app connects as a member of app_rw (SELECT/INSERT only).
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/djmagro/outlays/core/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx" for goose
	"github.com/pressly/goose/v3"
)

// Connect opens a pgx pool to the given Postgres URL and verifies it with a bounded ping.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// Migrate runs all goose Up migrations against ownerURL (the migration/owner role).
func Migrate(ownerURL string) error {
	db, err := sql.Open("pgx", ownerURL)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("dialect: %w", err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
