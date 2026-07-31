# Style Guide

This guide supplements standard Go conventions with repository-specific choices.

## General principle

Prefer code that can be understood locally. Explicit code is usually better than
a reusable abstraction that forces readers to jump between packages.

## Formatting and tools

Every Go change must pass:

```sh
gofmt -w .
go build ./...
go vet ./...
go test -race ./...
golangci-lint run
```

Do not manually align code against `gofmt`.

## Package naming

Use short lowercase names based on business language:

```text
subscriber
appointment
customer
billing
readiness
```

Avoid:

```text
subscriber_service
appointmentsFeature
common
helpers
utils
manager
```

Use one package per feature until there is a demonstrated reason to split.

## Type naming

Within a feature, prefer non-stuttering names:

```go
subscriber.Service
subscriber.Store
subscriber.SQLiteStore
subscriber.Input
```

Avoid:

```go
subscriber.SubscriberService
subscriber.SubscriberRepositoryInterface
subscriber.SubscriberManager
```

Use `Service` for the feature's use-case entrypoint when that concept fits. Do
not force every feature to have a service.

Use `Store` for a feature-owned persistence capability only when it represents
storage. Name other ports after their role:

```go
type Mailer interface { ... }
type PaymentGateway interface { ... }
type SubscriberReader interface { ... }
```

## Function and method naming

Use domain verbs:

```go
Subscribe
Book
Cancel
Activate
ListRecent
FindByEmail
```

Avoid generic CRUD words when a business operation exists:

```go
Create
Update
Process
Handle
Do
Execute
```

HTTP-only functions may use transport verbs such as `create`, `list`, or `show`
when they remain unexported and clearly map routes.

Constructors use `New` or a precise adapter name:

```go
subscriber.New(store, clock)
subscriber.NewSQLiteStore(db)
```

## Files

Name files by concern, not by arbitrary layer numbering:

```text
subscriber.go
port.go
http.go
sqlite.go
validation.go
transition.go
```

Tests follow their subject:

```text
subscriber_test.go
http_test.go
sqlite_test.go
```

Do not create `types.go`, `helpers.go`, or `utils.go` as dumping grounds. Name the
actual concept.

## Exports

Keep the exported surface minimal.

Export only what another package must construct or call. Co-location in one
feature package allows most implementation details to remain private.

Every exported declaration should have a useful doc comment unless its meaning
is completely conventional and lint policy permits omission.

## Interfaces

Define interfaces at the consumer.

Keep them narrow and behavior-oriented:

```go
type Store interface {
    Insert(context.Context, Subscriber) error
}
```

Avoid provider-wide interfaces:

```go
type Database interface {
    Exec(...)
    Query(...)
    Begin(...)
    Ping(...)
    Close(...)
}
```

Avoid generic CRUD repositories. They hide business semantics and usually grow
methods unused by individual consumers.

Accept interfaces; return concrete types, unless a caller has a real reason to
own an interface.

## Context

- `context.Context` is the first parameter.
- Pass it across I/O boundaries.
- Do not use context for optional function arguments.
- Do not store it on long-lived structs.
- Use private key types for request-scoped values.
- Respect cancellation; do not replace request context with
  `context.Background()` inside normal request work.

## Errors

Use sentinel errors only for stable classifications:

```go
var ErrInvalidEmail = errors.New("invalid email")
```

Wrap with useful operation context:

```go
return fmt.Errorf("insert subscriber: %w", err)
```

Classify with `errors.Is` and `errors.As`. Never parse error strings.

Error text should be lowercase and should not end with punctuation when it is
intended for wrapping.

The HTTP layer maps feature errors to status codes. Storage adapters translate
driver details before they escape.

## Validation

Business validation belongs to the feature service/use case. Transport parsing
may reject malformed representations earlier, but must not become the only
source of domain truth.

Prefer small named validation functions when they encode a meaningful invariant.
Avoid validation frameworks for simple Go structs.

Use constants or functions as the single source for important bounds. Do not
copy business constants between HTTP, SQL, and presentation files.

## Time and randomness

Use `time.Time` in domain APIs where a timestamp is meaningful. Normalize
location/precision deliberately when persistence or comparison requires it.

Inject clocks or generators when behavior depends on the current time or random
values:

```go
type Clock func() time.Time
```

Do not call `time.Now()` throughout business code if tests need deterministic
outcomes.

## HTTP

Use Go 1.22+ method-aware patterns:

```go
mux.HandleFunc("POST /subscribers", create(service))
```

Handlers should be shallow:

```text
parse → call service → map result → respond
```

Use explicit response content types and status codes. Do not expose raw internal
errors.

For JSON APIs, use stable response shapes. For larger inputs, limit body size,
validate content type, and consider rejecting unknown fields.

Do not write a response in multiple layers. Once headers/body are committed,
return.

## SQL

Use uppercase SQL keywords and readable multiline strings:

```go
const query = `
    INSERT INTO subscribers (id, email, created_at)
    VALUES (?, ?, ?)
`
```

Use parameter placeholders. Never construct SQL from untrusted strings.

Select explicit columns. Scan in the same order. Keep query and scan close.

Name columns and tables in `snake_case`. Prefer plural table names consistently
with the existing schema.

Let constraints enforce invariants such as uniqueness and foreign keys, while
keeping user-friendly validation at the feature boundary.

## Transactions

Put atomicity around the complete invariant, not only around individual writes.
Keep the transaction callback small. Return errors so the shared helper can
rollback.

Do not perform slow network calls while holding a database transaction unless
the design explicitly requires it.

## Logging

Use `slog` structured attributes:

```go
slog.Info("request", "method", r.Method, "status", status)
```

Use stable event messages and keys. Do not format structured values into one
large string.

Never log passwords, auth headers, tokens, cookies, or unnecessary personal
data.

## Comments

Explain why, invariants, ordering, and non-obvious tradeoffs.

Good:

```go
// accessLog stays outside recoverPanic so panics are recorded with status 500.
```

Avoid restating syntax:

```go
// Increment count.
count++
```

## Generics

Do not introduce generics preemptively. Use them when several real call sites
share the same type-safe algorithm and the generic version is easier to read.

Good possible uses:

- pure collection transformations;
- repeated scanning/helper mechanics with stable shapes;
- test helpers where type safety improves clarity.

Poor uses:

- universal repositories;
- generic service layers;
- abstractions that erase domain verbs.

## Dependencies

Prefer the standard library when it provides a clear solution. Add a dependency
only when it materially reduces risk or complexity.

Before adding one, check:

- maintenance and release health;
- license;
- transitive dependency cost;
- security history;
- whether a small local implementation is clearer;
- whether the dependency becomes part of the public architecture.

Document major dependency choices in `docs/DECISIONS.md`.

## Frontend

Use semantic HTML and accessible controls. Preserve keyboard navigation, focus,
labels, and contrast.

Use the CSS tokens as primitives. Do not turn the template into a mandatory
frontend stack.

Escape untrusted output using the selected template engine's safe defaults.
Avoid manual HTML string construction.
