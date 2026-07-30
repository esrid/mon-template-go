package readiness

import "context"

// Store is the persistence capability this feature needs — nothing more.
//
// The port lives with its consumer, never in a shared ports/ package: that is
// what keeps features from coupling to each other through a common interface
// dump. *sqlite.DB happens to satisfy it as-is, so internal/app wires them with
// no glue code. A Postgres or in-memory implementation swaps in the same way.
type Store interface {
	Ping(context.Context) error
}
