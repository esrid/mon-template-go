package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/samber/oops"
	modernsqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// WithinTransaction runs fn in a transaction, rolling back on error or panic.
func (d *DB) WithinTransaction(ctx context.Context, fn func(*sql.Tx) error) (err error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return oops.Code("db_begin_failed").Wrap(err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		} else if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return oops.Code("db_commit_failed").Wrap(err)
	}
	return nil
}

// DecorateError attaches the SQLite error code to err for structured logging.
func DecorateError(err error, operation string) error {
	if err == nil {
		return nil
	}
	wrapped := oops.Code("database_error").With("op", operation)
	if sqliteErr, ok := errors.AsType[*modernsqlite.Error](err); ok {
		return wrapped.With("sqlite_code", sqliteErr.Code()).Wrap(err)
	}
	return wrapped.Wrap(err)
}

// IsUniqueViolation reports whether err is a UNIQUE or PRIMARY KEY conflict,
// so a feature can map it to its own "already exists" error.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if sqliteErr, ok := errors.AsType[*modernsqlite.Error](err); ok {
		return sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE ||
			sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
