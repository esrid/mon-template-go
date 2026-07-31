# Architecture

## Purpose

`mon-template-go` is a small, production-oriented Go web template built as a
**feature-first modular monolith**.

Its goal is not architectural purity. Its goal is to keep the most common work
obvious:

```text
I am changing subscriber behavior
→ internal/feature/subscriber

I am changing database connection policy
→ internal/platform/sqlite

I am changing application wiring
→ internal/app
```

A developer should not need to search through global service, controller,
repository, and model directories to understand one capability.

## Architectural style

The template combines two ideas:

1. **Django-style feature ownership**: each capability owns its use case, HTTP
   entrypoints, ports, and persistence adapter in one package.
2. **Hexagonal dependency boundaries**: business code depends on interfaces it
   owns, while concrete infrastructure is selected at the composition root.

The directories are optimized for navigation; the interfaces preserve
replaceability and testability.

## Top-level structure

```text
cmd/web/                     process entrypoint
internal/app/                composition root and lifecycle
internal/feature/            vertical business slices
internal/platform/           shared technical plumbing
assets/                      optional frontend foundation
docs/                        engineering documentation
```

## Dependency map

```text
                        ┌────────────────────────┐
                        │        cmd/web         │
                        └───────────┬────────────┘
                                    │
                                    v
                        ┌────────────────────────┐
                        │      internal/app      │
                        │ wiring, routes, server │
                        └──────┬─────────┬───────┘
                               │         │
                     constructs│         │opens
                               v         v
              ┌──────────────────┐   ┌──────────────────┐
              │ feature packages │   │ platform packages │
              │ use cases + ports│   │ SQLite, config, web│
              └────────┬─────────┘   └──────────────────┘
                       │
                       │ calls feature-owned interfaces
                       v
              ┌──────────────────┐
              │ concrete adapter │
              │ e.g. SQLite store│
              └──────────────────┘
```

`internal/app` is the only package that deliberately sees the whole system.
Feature packages do not import one another, and platform packages do not import
features.

## The feature package

A feature is a complete vertical slice kept in one Go package.

The subscriber reference feature contains:

```text
internal/feature/subscriber/
  subscriber.go       types, validation, use cases, feature errors
  port.go             interfaces consumed by the feature
  http.go             routes and transport translation
  sqlite.go           SQLite implementation of the feature's store port
  *_test.go           business, HTTP, and SQLite tests
```

This co-location is deliberate. Splitting a small feature into separate
`domain`, `application`, `ports`, and `adapters` packages would force internal
details to become exported and would make one change require more navigation.

Split a feature only after the package is genuinely difficult to understand,
not because a diagram contains more boxes.

## Responsibilities inside a feature

### Business file

The main feature file owns:

- domain types;
- use-case orchestration;
- invariant validation;
- feature errors;
- business normalization;
- rules that must hold regardless of HTTP or SQLite.

It must not know about HTTP status codes, SQL syntax, Goose, environment
variables, or the process lifecycle.

### `port.go`

A port describes a capability the feature consumes.

```go
type Store interface {
    Insert(context.Context, Subscriber) error
}

type Clock func() time.Time
```

The interface belongs to the consumer because the consumer knows the smallest
contract it needs. A central interface package tends to become a catalogue of
provider-shaped APIs and couples unrelated features.

Not every dependency needs an interface. Pure helpers and stable internal
implementation details should remain concrete.

### `http.go`

The HTTP adapter:

1. registers complete route patterns;
2. parses request data;
3. calls the service;
4. maps feature errors to HTTP responses;
5. serializes the response.

It does not decide domain policy. Duplicate parsing checks are acceptable when
they protect a transport boundary, but the authoritative rule must remain in the
feature service.

### `sqlite.go`

The feature owns the SQL required to implement its port. This keeps feature
queries next to the behavior they support.

The SQLite adapter may:

- execute parameterized SQL;
- scan rows;
- use transactions;
- translate SQLite constraint failures;
- return feature-level classifications.

It must not own algorithms that can run independently of SQLite.

## Platform packages

`internal/platform` contains infrastructure shared across features.

### `config`

Loads environment configuration and validates operational limits. Feature code
does not call `os.Getenv` directly.

### `sqlite`

Owns:

- opening and closing the shared database;
- SQLite pragmas and connection policy;
- embedded global Goose migrations;
- transaction helpers;
- generic SQLite error classification.

It does not own feature queries.

### `web`

Owns generic HTTP mechanics:

- request IDs;
- access logging;
- panic recovery;
- baseline security headers;
- common response helpers.

It does not know which features exist.

## Composition root

`internal/app` is the equivalent of a Django project's root URL and application
configuration.

`app.go`:

- validates configuration;
- opens concrete infrastructure;
- constructs each service;
- injects clocks and stores;
- configures the HTTP server;
- owns shutdown and resource closure.

`routes.go`:

- creates the root `ServeMux`;
- mounts each feature once;
- wraps the final router in shared middleware.

Neither file contains feature behavior.

## Request flow

For the subscriber slice:

```text
HTTP request
    ↓
subscriber/http.go
    ↓ parse transport input
subscriber.Service
    ↓ enforce business rules
subscriber.Store (feature-owned port)
    ↓
subscriber.SQLiteStore
    ↓ execute SQL
platform/sqlite.DB
    ↓
SQLite
```

Tests can stop at any boundary:

- service test injects a store stub and clock;
- HTTP test exercises transport mapping;
- SQLite test uses a real temporary/in-memory database;
- app test verifies construction, routes, and middleware.

## Cross-feature collaboration

Feature packages never import one another.

When feature A needs a capability supplied by feature B:

1. A declares a narrow interface in A's package.
2. B exposes a method or small adapter that satisfies it.
3. `internal/app` injects B into A.

Example:

```go
// package dashboard
type SubscriberReader interface {
    Recent(context.Context, int) ([]SubscriberSummary, error)
}
```

The dashboard does not need to know B's storage implementation or internal
service shape. This preserves feature ownership and prevents import cycles.

Use this pattern for real collaboration only. Do not create interfaces in
advance for hypothetical relationships.

## Routes

Each feature owns its full route paths:

```go
func Mount(mux *http.ServeMux, service *Service) {
    mux.HandleFunc("POST /subscribers", create(service))
}
```

`internal/app/routes.go` does not prefix or reinterpret those paths. This makes a
feature's public surface visible from the feature itself.

No feature registers `/` as a fallback. The root router owns unknown-route 404s.

## Persistence and migrations

Feature SQL is local, but migrations are globally ordered:

```text
internal/platform/sqlite/migrations/
  00001_init.sql
  00002_subscribers.sql
```

This is intentional. Goose maintains one version sequence and does not resolve a
Django-style dependency graph across per-feature migration directories.

A new feature therefore:

- adds its SQL adapter in the feature package;
- adds its schema migration to the global migration sequence;
- proves both with feature integration tests.

## Replacing SQLite

Feature business code does not import SQLite libraries. A different adapter can
satisfy the same feature-owned ports.

A PostgreSQL migration is not necessarily a one-line change because SQL adapters
and migrations are database-specific, but the service and HTTP layers should
remain stable.

The expected path is:

```text
internal/platform/postgres/       pool and migration infrastructure
internal/feature/subscriber/postgres.go
internal/app/app.go                selects the new adapter
```

Do not create a fake universal database abstraction that hides meaningful SQL
and transaction differences.

## Scaling the codebase

This structure scales by adding vertical slices:

```text
internal/feature/
  identity/
  customer/
  appointment/
  conversation/
  followup/
  dashboard/
```

Keep business names. Avoid organizing growth into global technical buckets such
as `handlers/`, `repositories/`, or `services/`.

A feature may be split when multiple signals are present:

- navigation is consistently painful;
- unrelated responsibilities change for different reasons;
- the package has clear internal boundaries;
- tests and exports improve rather than merely move;
- the split does not create circular dependencies.

Line count alone is not a sufficient reason.

## What belongs outside the base template

The template intentionally does not choose:

- authentication/session implementation;
- CSRF and CORS policy;
- frontend framework;
- mail provider;
- queue;
- cache;
- payments;
- object storage;
- observability vendor;
- multi-tenancy model.

These decisions are product-specific. Add them as platform implementations and
feature-owned ports when a concrete project needs them.

## Architectural tests by inspection

A change fits the architecture when these answers are obvious:

- Which feature owns the behavior?
- Can its rules be tested without HTTP and SQLite?
- Does the feature declare only the dependencies it consumes?
- Is concrete infrastructure selected in `internal/app`?
- Are feature queries local to the feature?
- Did shared platform code remain free of business language?
- Can a developer find the complete capability in one directory?

When those answers become unclear, improve ownership before adding more layers.
