package subscriber

import (
	"context"
	"time"
)

// Store is the persistence this feature needs. Implementations must translate
// storage errors into the feature's own: a duplicate email is ErrDuplicate, a
// missing row is ErrNotFound. That is what keeps driver types out of the
// use-case and the handlers.
type Store interface {
	Create(ctx context.Context, email string, createdAt time.Time) (Subscriber, error)
	List(ctx context.Context) ([]Subscriber, error)
	ByID(ctx context.Context, id int64) (Subscriber, error)
}

// Clock is a port too. Ports are not only for databases: any ambient input —
// the wall clock, a mailer, a payment gateway — enters the feature as an
// interface it declares itself.
type Clock func() time.Time
