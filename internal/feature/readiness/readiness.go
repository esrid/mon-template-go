// Package readiness answers one question: can this process serve traffic?
//
// It is the template's reference feature. A feature is a vertical slice: its
// use-case, the interfaces it needs, its HTTP routes, and its SQL all live in
// this one package. Keeping it a single package is deliberate — splitting a
// feature into domain/, service/ and store/ subpackages would force every
// internal type to be exported, which destroys the encapsulation the split was
// meant to create.
package readiness

import (
	"context"
	"fmt"
)

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Check(ctx context.Context) error {
	if err := s.store.Ping(ctx); err != nil {
		return fmt.Errorf("readiness: persistence: %w", err)
	}
	return nil
}
