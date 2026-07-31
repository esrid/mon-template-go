# mon-template-go

A small Go web application template built from self-contained feature modules,
with a standard-library HTTP server, SQLite/WAL, embedded Goose migrations,
manual dependency injection, tests, and a production container.


## Engineering documentation

- [`AGENTS.md`](AGENTS.md) — mandatory rules for coding agents and contributors
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — architecture and dependency model
- [`docs/PROJECT_STRUCTURE.md`](docs/PROJECT_STRUCTURE.md) — directory placement guide
- [`docs/STYLE_GUIDE.md`](docs/STYLE_GUIDE.md) — Go, HTTP, SQL, and naming conventions
- [`docs/SECURITY.md`](docs/SECURITY.md) — security baseline and project additions
- [`docs/TESTING.md`](docs/TESTING.md) — test strategy and examples
- [`docs/CODE_REVIEW.md`](docs/CODE_REVIEW.md) — pre-merge checklist
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — architectural decision log

## Start

```sh
make run
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

Or run the container:

```sh
docker compose up --build
```

## Layout

The code is cut by feature first, by layer second. A feature is a vertical
slice you can read, test, or delete on its own.

```text
cmd/web/                     process entrypoint and signal handling
internal/
  app/                       composition root: builds features, mounts routes
  platform/                  technical plumbing, no business logic
    config/                  environment configuration and validation
    sqlite/                  connection, pragmas, migrations, tx helper
    web/                     middleware and response helpers
  feature/
    readiness/               smallest slice: use-case, port, routes
    subscriber/              full slice — copy this one for real features
      subscriber.go          use-case, validation, its own errors
      port.go                the interfaces it needs (Store, Clock)
      http.go                routes, JSON, error-to-status mapping
      sqlite.go              its SQL, behind the port above
assets/                      optional frontend sources; no toolchain imposed
```

Three rules keep this readable as it grows:

1. **Ports live with the feature that consumes them**, never in a shared
   `ports/` package — that is what stops features coupling to each other.
2. **One package per feature.** Splitting a feature into `domain/`, `service/`
   and `store/` subpackages would force its internals to be exported, which
   destroys the encapsulation the split was meant to create. Split only when a
   feature genuinely hurts.
3. **Features never import each other.** They meet in `internal/app`.

### Adding a feature

Copy `internal/feature/subscriber/` and rename. It shows the whole shape:
validation, feature-owned errors mapped to status codes, its own SQL, and an
injected clock so the use-case stays deterministic.

```text
internal/feature/billing/
  billing.go     use-case and its types
  port.go        the interfaces it needs (Store, Mailer, PaymentGateway…)
  http.go        func Mount(*http.ServeMux, *Service)
  sqlite.go      the SQL, behind the port above
```

Ports are not only for databases. Any ambient or external input — the clock, a
mailer, a payment gateway — enters as an interface the feature declares itself,
so tests inject a stub and production wires the real one in `internal/app`.

Each feature declares its own full paths, like Django's `include()`:

```go
func Mount(mux *http.ServeMux, service *Service) {
	mux.HandleFunc("GET /billing/invoices", list(service))
	mux.HandleFunc("GET /billing/invoices/{id}", show(service))
}
```

Then build it in `internal/app/app.go` and mount it in `internal/app/routes.go`:

```go
billing.Mount(root, a.billing)
```

No feature registers `"/"`, so unknown paths get the root router's 404 instead
of one that belongs to an unrelated feature.

The optional frontend foundation includes neutral, accessible light/dark design
tokens in `assets/src/css/tokens.css`. See `assets/src/css/README.md` for usage
and brand customization; no reset, components, or CSS framework are imposed.

## Swapping external services

`internal/platform/sqlite` is a concrete adapter. No feature imports
`database/sql`, the SQLite driver, or Goose through anything but its own port.

`*sqlite.DB` satisfies `readiness.Store` structurally, so `internal/app` wires
them with no glue code. To move to PostgreSQL, add `internal/platform/postgres`
satisfying the same feature-owned ports and change the one construction line in
`internal/app/app.go`. Features and handlers stay untouched. The same shape
applies to any external service — mail, payments, object storage: the feature
declares the narrow interface it needs, `platform/` implements it, `app/` wires
it.

Migrations live in `internal/platform/sqlite/migrations/` and run when the
database opens. A feature owns its tables and its SQL, but not its migration
file — name the file after the feature (`00002_subscribers.sql`) and point to it
from the feature's `sqlite.go`.

They stay centralised on purpose. Django can give each app its own migrations
because it resolves them through a dependency graph; goose has a single global
version sequence and no such graph, so per-feature directories would make the
order between features depend on the wiring order in `internal/app` — implicit,
and silent when a cross-feature foreign key applies too early. A feature copied
without its migration fails loudly in that feature's own tests, which is the
guard that makes the trade-off cheap.

If parallel branches start colliding on version numbers, switch to timestamped
versions and `goose.WithAllowOutofOrder` before splitting the directory.

The in-memory DSN is restricted to one connection so its schema remains
consistent. File databases use WAL, foreign keys, a busy timeout, and a small
connection pool.

## Configuration

| Variable                   | Default  | Purpose                               |
| -------------------------- | -------- | ------------------------------------- |
| `HTTP_ADDR`                | `:8080`  | HTTP listen address                   |
| `DATABASE_DSN`             | `app.db` | SQLite DSN (`DSN` remains a fallback) |
| `HTTP_MAX_HEADER_BYTES`    | `65536`  | Maximum request-header size           |
| `HTTP_READ_HEADER_TIMEOUT` | `5s`     | Request-header deadline               |
| `HTTP_READ_TIMEOUT`        | `15s`    | Full-request read deadline            |
| `HTTP_WRITE_TIMEOUT`       | `30s`    | Response write deadline               |
| `HTTP_IDLE_TIMEOUT`        | `60s`    | Keep-alive idle deadline              |
| `SHUTDOWN_TIMEOUT`         | `10s`    | Graceful shutdown deadline            |

## Commands

```sh
make build
make run
make test
make vet
make lint     # requires golangci-lint
make vuln     # checks reachable known vulnerabilities
```

Tests cover configuration, middleware, both features end to end (validation,
error-to-status mapping, SQL against a real in-memory database), route mounting
and wiring, SQLite migrations, connection policy, transactions, and constraint
detection.

## Use as a template

After creating a repository, replace `github.com/esrid/mon-template-go` in
`go.mod` and Go imports with your module path, then run:

```sh
go mod tidy
go test ./...
```

Authentication, sessions, CSRF/CORS policy, queues, email, caching, object
storage, payments, observability vendors, and frontend frameworks are
deliberately project-specific.
