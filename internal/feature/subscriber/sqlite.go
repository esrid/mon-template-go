package subscriber

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/esrid/mon-template-go/internal/platform/sqlite"
)

// SQLiteStore implements Store. The SQL lives with the feature that owns the
// table; internal/platform/sqlite only supplies the connection and helpers.
type SQLiteStore struct {
	db *sqlite.DB
}

func NewSQLiteStore(db *sqlite.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Create(ctx context.Context, email string, createdAt time.Time) (Subscriber, error) {
	row := s.db.SQL().QueryRowContext(ctx,
		`INSERT INTO subscribers (email, created_at) VALUES (?, ?)
		 RETURNING id, email, created_at`,
		email, createdAt.Unix(),
	)
	created, err := scan(row.Scan)
	if err != nil {
		if sqlite.IsUniqueViolation(err) {
			return Subscriber{}, ErrDuplicate
		}
		return Subscriber{}, sqlite.DecorateError(err, "subscriber.Create")
	}
	return created, nil
}

func (s *SQLiteStore) List(ctx context.Context) ([]Subscriber, error) {
	rows, err := s.db.SQL().QueryContext(ctx,
		`SELECT id, email, created_at FROM subscribers ORDER BY id`)
	if err != nil {
		return nil, sqlite.DecorateError(err, "subscriber.List")
	}
	defer func() { _ = rows.Close() }()

	subscribers := make([]Subscriber, 0)
	for rows.Next() {
		found, err := scan(rows.Scan)
		if err != nil {
			return nil, sqlite.DecorateError(err, "subscriber.List")
		}
		subscribers = append(subscribers, found)
	}
	if err := rows.Err(); err != nil {
		return nil, sqlite.DecorateError(err, "subscriber.List")
	}
	return subscribers, nil
}

func (s *SQLiteStore) ByID(ctx context.Context, id int64) (Subscriber, error) {
	row := s.db.SQL().QueryRowContext(ctx,
		`SELECT id, email, created_at FROM subscribers WHERE id = ?`, id)
	found, err := scan(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Subscriber{}, ErrNotFound
		}
		return Subscriber{}, sqlite.DecorateError(err, "subscriber.ByID")
	}
	return found, nil
}

// scan reads one row in the column order every query above selects.
func scan(into func(...any) error) (Subscriber, error) {
	var (
		found     Subscriber
		createdAt int64
	)
	if err := into(&found.ID, &found.Email, &createdAt); err != nil {
		return Subscriber{}, err
	}
	found.CreatedAt = time.Unix(createdAt, 0).UTC()
	return found, nil
}
