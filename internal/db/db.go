// Package db holds the Postgres connection, the migration runner and the
// queries behind the API.
package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// DB wraps a pgx connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// Connect opens a pool and waits until the database answers or timeout
// elapses (compose may start the API before Postgres is ready).
func Connect(ctx context.Context, url string, timeout time.Duration) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("db: parse DATABASE_URL: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: create pool: %w", err)
	}
	deadline := time.Now().Add(timeout)
	for {
		pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = pool.Ping(pctx)
		cancel()
		if err == nil {
			return &DB{Pool: pool}, nil
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			pool.Close()
			return nil, fmt.Errorf("db: database not reachable after %s: %w", timeout, err)
		}
		slog.Info("waiting for database", "err", err)
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
		}
	}
}

// Close closes the pool.
func (d *DB) Close() { d.Pool.Close() }
