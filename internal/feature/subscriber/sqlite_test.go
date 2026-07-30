package subscriber

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/esrid/mon-template-go/internal/platform/sqlite"
)

func testStore(t *testing.T) *SQLiteStore {
	t.Helper()
	db, err := sqlite.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewSQLiteStore(db)
}

func TestCreateReturnsTheStoredRow(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)

	created, err := store.Create(ctx, "bob@example.com", frozen)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == 0 {
		t.Fatal("Create() returned a zero id")
	}
	if created.Email != "bob@example.com" {
		t.Fatalf("Email = %q", created.Email)
	}
	// A round-trip through the INTEGER column must not shift the timestamp.
	if !created.CreatedAt.Equal(frozen) {
		t.Fatalf("CreatedAt = %v, want %v", created.CreatedAt, frozen)
	}

	read, err := store.ByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("ByID() error = %v", err)
	}
	if read != created {
		t.Fatalf("ByID() = %+v, want %+v", read, created)
	}
}

func TestCreateMapsUniqueViolationToErrDuplicate(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)

	if _, err := store.Create(ctx, "bob@example.com", frozen); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	_, err := store.Create(ctx, "bob@example.com", frozen.Add(time.Hour))
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("second Create() error = %v, want ErrDuplicate", err)
	}
}

func TestByIDMapsMissingRowToErrNotFound(t *testing.T) {
	_, err := testStore(t).ByID(context.Background(), 404)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ByID() error = %v, want ErrNotFound", err)
	}
}

func TestListIsOrderedAndNeverNil(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)

	empty, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	// Must be an empty slice, not nil: json.Marshal renders nil as "null".
	if empty == nil {
		t.Fatal("List() returned nil on an empty table")
	}

	for _, email := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		if _, err := store.Create(ctx, email, frozen); err != nil {
			t.Fatalf("Create(%q) error = %v", email, err)
		}
	}

	all, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("List() length = %d, want 3", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ID >= all[i].ID {
			t.Fatalf("List() is not ordered by id: %+v", all)
		}
	}
}
