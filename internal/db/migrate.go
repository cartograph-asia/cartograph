package db

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// migrationLockID is an arbitrary constant for pg_advisory_lock so that only
// one process applies migrations at a time.
const migrationLockID = 727_001_008

// Migrate applies every *.sql file in fsys that is not yet recorded in
// schema_migrations, in lexical order, each in its own transaction.
// It is safe to call on every start-up.
func (d *DB) Migrate(ctx context.Context, fsys fs.FS) error {
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return fmt.Errorf("migrate: list files: %w", err)
	}
	sort.Strings(names)

	conn, err := d.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("migrate: acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "select pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("migrate: take advisory lock: %w", err)
	}
	defer func() {
		// Use a fresh context so the lock is released even if ctx is cancelled.
		_, _ = conn.Exec(context.WithoutCancel(ctx), "select pg_advisory_unlock($1)", migrationLockID)
	}()

	if _, err := conn.Exec(ctx, `create table if not exists schema_migrations (
		version    text primary key,
		applied_at timestamptz not null default now()
	)`); err != nil {
		return fmt.Errorf("migrate: create schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := conn.Query(ctx, "select version from schema_migrations")
	if err != nil {
		return fmt.Errorf("migrate: read schema_migrations: %w", err)
	}
	versions, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("migrate: read schema_migrations: %w", err)
	}
	for _, v := range versions {
		applied[v] = true
	}

	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		if applied[version] {
			continue
		}
		sql, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", name, err)
		}
		err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			// No arguments: pgx uses the simple protocol, so a file may hold
			// several statements.
			if _, err := tx.Exec(ctx, string(sql)); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, "insert into schema_migrations (version) values ($1)", version)
			return err
		})
		if err != nil {
			return fmt.Errorf("migrate: apply %s: %w", name, err)
		}
		slog.Info("applied migration", "version", version)
	}
	return nil
}
