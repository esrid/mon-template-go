package subscriber

import (
	"context"
	"errors"
	"testing"
	"time"
)

// frozen is the injected clock: Subscribe stays deterministic, so the test
// asserts an exact timestamp instead of a tolerance window.
var frozen = time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)

type storeStub struct {
	created   Subscriber
	createErr error
	gotEmail  string
	gotTime   time.Time
}

func (s *storeStub) Create(_ context.Context, email string, createdAt time.Time) (Subscriber, error) {
	s.gotEmail, s.gotTime = email, createdAt
	if s.createErr != nil {
		return Subscriber{}, s.createErr
	}
	return s.created, nil
}

func (s *storeStub) List(context.Context) ([]Subscriber, error) { return nil, nil }

func (s *storeStub) ByID(context.Context, int64) (Subscriber, error) {
	return Subscriber{}, ErrNotFound
}

func TestSubscribeNormalisesTheAddress(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lower-cased", "Bob@Example.COM", "bob@example.com"},
		{"surrounding spaces trimmed", "  bob@example.com  ", "bob@example.com"},
		{"display name dropped", "Bob <bob@example.com>", "bob@example.com"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &storeStub{}
			service := New(store, func() time.Time { return frozen })

			if _, err := service.Subscribe(context.Background(), test.input); err != nil {
				t.Fatalf("Subscribe(%q) error = %v", test.input, err)
			}
			if store.gotEmail != test.want {
				t.Fatalf("stored email = %q, want %q", store.gotEmail, test.want)
			}
			if !store.gotTime.Equal(frozen) {
				t.Fatalf("stored time = %v, want %v", store.gotTime, frozen)
			}
		})
	}
}

func TestSubscribeRejectsInvalidAddresses(t *testing.T) {
	for _, input := range []string{"", "   ", "not-an-email", "a@", "@example.com", "a b@example.com"} {
		t.Run(input, func(t *testing.T) {
			service := New(&storeStub{}, func() time.Time { return frozen })
			_, err := service.Subscribe(context.Background(), input)
			if !errors.Is(err, ErrInvalidEmail) {
				t.Fatalf("Subscribe(%q) error = %v, want ErrInvalidEmail", input, err)
			}
		})
	}
}

func TestSubscribePropagatesStoreErrors(t *testing.T) {
	service := New(&storeStub{createErr: ErrDuplicate}, func() time.Time { return frozen })
	_, err := service.Subscribe(context.Background(), "bob@example.com")
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("Subscribe() error = %v, want ErrDuplicate", err)
	}
}

func TestByIDRejectsNonPositiveIDs(t *testing.T) {
	service := New(&storeStub{}, nil)
	for _, id := range []int64{0, -1} {
		if _, err := service.ByID(context.Background(), id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("ByID(%d) error = %v, want ErrNotFound", id, err)
		}
	}
}

func TestNilClockFallsBackToTimeNow(t *testing.T) {
	store := &storeStub{}
	before := time.Now().UTC().Truncate(time.Second)

	if _, err := New(store, nil).Subscribe(context.Background(), "bob@example.com"); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if store.gotTime.Before(before) {
		t.Fatalf("stored time = %v, want >= %v", store.gotTime, before)
	}
}
