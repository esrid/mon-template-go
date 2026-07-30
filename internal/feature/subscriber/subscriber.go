// Package subscriber manages a mailing list: sign-ups and lookups.
//
// It is the template's second vertical slice, and the one to copy for a real
// feature: it owns its validation, its ports, its SQL, and its routes.
package subscriber

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

type Subscriber struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Errors this feature defines. Handlers map them to status codes; nothing
// outside the feature needs to know how they are stored.
var (
	ErrInvalidEmail = errors.New("subscriber: email is not a valid address")
	ErrDuplicate    = errors.New("subscriber: email is already subscribed")
	ErrNotFound     = errors.New("subscriber: not found")
)

type Service struct {
	store Store
	now   Clock
}

// New builds the service. Passing now explicitly keeps Subscribe deterministic:
// the only ambient input is injected, so tests assert an exact timestamp.
// A nil clock falls back to time.Now.
func New(store Store, now Clock) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, now: now}
}

// Subscribe validates and normalises an address, then records it.
// It returns ErrInvalidEmail or ErrDuplicate for expected rejections.
func (s *Service) Subscribe(ctx context.Context, email string) (Subscriber, error) {
	address, err := normaliseEmail(email)
	if err != nil {
		return Subscriber{}, err
	}
	// Second precision: the stored column is a Unix timestamp, so truncating
	// here keeps what the caller receives equal to what a later read returns.
	return s.store.Create(ctx, address, s.now().UTC().Truncate(time.Second))
}

func (s *Service) List(ctx context.Context) ([]Subscriber, error) {
	return s.store.List(ctx)
}

func (s *Service) ByID(ctx context.Context, id int64) (Subscriber, error) {
	if id <= 0 {
		return Subscriber{}, ErrNotFound
	}
	return s.store.ByID(ctx, id)
}

// normaliseEmail lower-cases the address so the UNIQUE index rejects
// case-variant duplicates.
//
// mail.ParseAddress follows RFC 5322: it accepts a display name ("Bob
// <bob@example.com>" yields bob@example.com) and does not require a dot in the
// domain. Deliverability is a confirmation email's job, not a regex's.
func normaliseEmail(raw string) (string, error) {
	address, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidEmail, err)
	}
	return strings.ToLower(address.Address), nil
}
