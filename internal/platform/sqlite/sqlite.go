// Package sqlite owns the SQLite connection, pragmas, and schema migrations.
// It holds no business logic: features build their own queries on top of *DB
// and expose them through interfaces they define themselves.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// Migrations are centralised here rather than split per feature: goose orders
// versions globally, so a single directory keeps cross-feature foreign keys
// safe. To move to per-feature migrations, give each feature its own provider
// with goose.WithTableName (available in goose v3.27.2) and accept that the
// ordering between features becomes the wiring order in internal/app.
//
//go:embed migrations/*.sql
var migrations embed.FS

type DB struct {
	db *sql.DB
}

func Open(ctx context.Context, dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", withPragmas(dsn))
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}

	maxConnections := 10
	if isMemoryDSN(dsn) {
		maxConnections = 1
	}
	db.SetMaxOpenConns(maxConnections)
	db.SetMaxIdleConns(maxConnections)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite: ping: %w", err)
	}
	if err := runMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

// SQL exposes the pool so a feature can write its own queries. Keep that SQL
// inside the feature package, behind an interface the feature owns.
func (d *DB) SQL() *sql.DB { return d.db }

func (d *DB) Ping(ctx context.Context) error { return d.db.PingContext(ctx) }

func (d *DB) Close() error { return d.db.Close() }

func runMigrations(ctx context.Context, db *sql.DB) error {
	migrationFS, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("sqlite: migration filesystem: %w", err)
	}
	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		db,
		migrationFS,
		goose.WithDisableGlobalRegistry(true),
		goose.WithLogger(goose.NopLogger()),
	)
	if err != nil {
		return fmt.Errorf("sqlite: migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("sqlite: migrations: %w", err)
	}
	return nil
}

func withPragmas(dsn string) string {
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + "_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
}

func isMemoryDSN(dsn string) bool {
	return dsn == ":memory:" || strings.Contains(dsn, "mode=memory")
}
