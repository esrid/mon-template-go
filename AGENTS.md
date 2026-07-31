# AGENTS.md

This file defines the operating rules for coding agents working in this repository.
It is normative: when a generated change conflicts with this file, follow this file
unless the task explicitly requires an architectural change.

## 1. Project intent

This repository is a small Go web application template optimized for:

- fast navigation;
- explicit dependencies;
- standard-library-first code;
- feature ownership;
- deterministic tests;
- production-safe defaults;
- easy replacement of infrastructure adapters.

The project is a **modular monolith organized by business feature**. It keeps the
useful dependency rules of Hexagonal Architecture without creating global
`domain/`, `services/`, `ports/`, `repositories/`, or `adapters/` trees.

## 2. Read before changing code

Before implementing a task:

1. Read `README.md`.
2. Read `docs/ARCHITECTURE.md`.
3. Inspect the nearest existing feature, usually
   `internal/feature/subscriber/`.
4. Inspect `internal/app/app.go` and `internal/app/routes.go` before changing
   construction or routing.
5. Run the relevant tests before and after the change.

Do not infer the architecture from generic Go conventions when this repository
already contains a local example.

## 3. Directory ownership

### `cmd/`

Process entrypoints only. Parse no business input and contain no business rules.
Entrypoints load configuration, establish process context, call `internal/app`,
and translate a fatal error into process exit.

### `internal/app/`

The composition root and application lifecycle. This is the only package allowed
to know both feature packages and concrete platform implementations.

It may:

- open infrastructure;
- construct services;
- inject dependencies;
- configure the HTTP server;
- mount feature routes;
- coordinate shutdown.

It must not contain business rules, SQL, request parsing, or feature-specific
response logic.

### `internal/feature/<name>/`

A vertical business slice. Keep a feature in one Go package until the package is
proven painful to navigate.

Typical files:

- `<feature>.go`: use cases, domain types, validation, feature errors;
- `port.go`: narrow interfaces consumed by the feature;
- `http.go`: routes and HTTP translation;
- `sqlite.go`: SQLite implementation of feature-owned ports;
- `*_test.go`: unit and integration tests.

A feature owns its use cases, language, errors, HTTP routes, and persistence
queries. It does not own the shared database connection, migration runner,
middleware stack, or process lifecycle.

### `internal/platform/`

Reusable technical plumbing that knows nothing about business features.
Examples: configuration, database connection policy, migrations, HTTP
middleware, response helpers.

Platform packages must never import feature packages.

### `assets/`

Optional frontend sources. No frontend toolchain is mandatory. Add a tool only
when a project needs it.

## 4. Dependency rules

Allowed direction:

```text
cmd -> app
app -> feature
app -> platform
feature -> platform/web only for generic HTTP helpers
platform -> standard library / external infrastructure libraries
```

Core rules:

1. Feature packages never import other feature packages.
2. Platform packages never import feature packages.
3. Features meet only in `internal/app` through dependency injection.
4. Interfaces are declared by the package that consumes them.
5. Do not create a shared `ports` package.
6. Do not make `internal/app` a service locator.
7. Do not use package globals for application dependencies.
8. Avoid `init` functions for registration or wiring.
9. Keep dependencies visible in constructors.
10. Prefer compile-time structural interface satisfaction; add assertions when
    they clarify a non-obvious adapter contract.

When a feature needs information supplied by another feature, declare the
smallest local interface representing that need and inject an implementation in
`internal/app`. Never import the neighboring feature merely to call its service.

## 5. Adding a feature

Use `internal/feature/subscriber/` as the full reference slice.

1. Create `internal/feature/<name>/`.
2. Put business types, validation, errors, and use cases in `<name>.go`.
3. Put dependency interfaces in `port.go`.
4. Add `http.go` only when the feature exposes HTTP routes.
5. Add `sqlite.go` only when the feature persists data.
6. Add a globally ordered migration under
   `internal/platform/sqlite/migrations/`.
7. Construct the service in `internal/app/app.go`.
8. Mount the routes in `internal/app/routes.go`.
9. Add unit tests and, where applicable, SQLite/HTTP integration tests.
10. Run formatting, build, vet, race tests, and lint.

Do not copy files a feature does not need.

## 6. Business logic

Business rules belong in the feature use-case code, not in transport or storage.

The following belong in feature business code:

- validation that must hold regardless of transport;
- state transitions;
- quotas and limits;
- idempotency semantics;
- scheduling or capacity algorithms;
- permissions expressed in business terms;
- normalization required by the domain;
- feature-owned error values.

The following do not belong in business code:

- HTTP status codes;
- JSON encoding;
- SQL syntax;
- SQLite error codes;
- environment-variable access;
- server lifecycle;
- template rendering details.

If an algorithm can run without SQLite, keep it out of `sqlite.go`.

## 7. Interfaces and ports

Declare an interface only at a real substitution boundary or when it improves a
consumer's tests.

Good candidates:

- persistence required by a service;
- clock or randomness;
- mail, payment, object storage, queues;
- a narrow read capability supplied by another feature at composition time.

Avoid interfaces for:

- every concrete type by default;
- internal helpers;
- one-off pure functions;
- speculative future providers;
- generic CRUD abstractions.

Prefer business vocabulary:

```go
type Store interface {
    Subscribe(context.Context, Subscriber) error
    FindByEmail(context.Context, string) (Subscriber, error)
}
```

Avoid flattening business behavior into `Create`, `Update`, `Delete`, and `All`
when more precise names exist.

Keep interfaces small. A consumer should not depend on methods it never calls.

## 8. Constructors and services

- Use constructors to make required dependencies explicit.
- Reject invalid configuration at the composition boundary.
- Inject clocks, generators, and external clients when determinism matters.
- Keep service fields private.
- Return concrete service types unless callers need an interface they own.
- Do not hide dependencies in context values.
- Pass `context.Context` as the first parameter for I/O-bound operations.
- Never store a request context on a long-lived service.

## 9. HTTP rules

This project uses `net/http` and Go's method-aware `ServeMux` patterns.

Each feature exposes one `Mount` function:

```go
func Mount(mux *http.ServeMux, service *Service)
```

Rules:

1. A feature declares its complete route paths.
2. No feature registers `/` merely to catch unknown paths.
3. `internal/app/routes.go` only mounts features and middleware.
4. Handlers parse transport input, call a use case, and translate the result.
5. Handlers do not decide business policy.
6. Map feature errors to HTTP status codes in the feature's HTTP layer.
7. Do not leak internal errors or SQL details to clients.
8. Set explicit content types.
9. Use `web.NoStore` for health endpoints and sensitive authenticated responses.
10. Keep request bodies bounded before decoding when accepting non-trivial input.
11. Reject unknown JSON fields when a strict API contract is desirable.
12. Validate media types for endpoints that require JSON or form input.
13. Respect request cancellation by passing `r.Context()` onward.
14. Do not log credentials, session tokens, authorization headers, or full
    sensitive request bodies.
15. Preserve the middleware order unless tests and documentation are updated.

Shared HTTP mechanics belong in `internal/platform/web`; feature-specific
mapping remains in the feature.

## 10. Error handling

- Use errors as values.
- Define stable feature errors with `errors.New` when callers must classify them.
- Wrap errors with operation context using `%w`.
- Check classifications with `errors.Is` / `errors.As`.
- Do not compare error strings.
- Do not discard errors silently, except best-effort response encoding where the
  headers have already been committed.
- Log an error once at the boundary responsible for operating the process or
  request; avoid duplicate logging at every layer.
- User-facing messages must not expose database details, stack traces, file
  paths, or secrets.

Use `oops` only when its structured context materially improves an error path;
do not wrap every error mechanically.

## 11. SQLite and SQL

SQLite infrastructure lives in `internal/platform/sqlite`. Feature SQL lives in
the feature's `sqlite.go`.

Rules:

1. Use parameterized queries; never interpolate untrusted data into SQL.
2. Pass context to database operations.
3. Keep transactions as short as correctness permits.
4. Use `WithinTransaction` for operations that must commit or roll back as a
   unit.
5. Check every `Scan`, `Exec`, `Query`, row iteration, and transaction error.
6. Close `Rows` and inspect `Rows.Err()`.
7. Translate driver-specific constraint errors inside the SQLite adapter.
8. Return feature errors or neutral wrapped errors to the service layer.
9. Preserve foreign keys, WAL, busy timeout, and current connection policy
   unless a measured requirement justifies changing them.
10. Do not put feature SQL into `internal/platform/sqlite`.
11. Do not put reusable business algorithms into `sqlite.go`.
12. Select explicit columns; avoid `SELECT *` in maintained application queries.
13. Keep migration SQL compatible with the SQLite version used by the project.
14. Index demonstrated lookup and constraint paths, not hypothetical ones.
15. Never rely only on application validation for uniqueness or referential
    integrity when the database can enforce it.

### Multi-tenant projects

The template is not multi-tenant by default. When a project adds tenancy:

- include tenant identity in every tenant-owned table key or constraint;
- scope every read and write by tenant;
- never load globally and filter in Go;
- use transactions/locking when tenant-scoped invariants require atomicity;
- add tests proving cross-tenant reads and writes fail;
- prefer APIs that require `tenantID` rather than APIs where it is optional.

## 12. Migrations

- Keep migrations in `internal/platform/sqlite/migrations/`.
- Use one global Goose sequence.
- Name migrations after the owning feature or schema change.
- Never edit an applied production migration; add a new one.
- Include reversible `Down` sections when reversal is safe and meaningful.
- Test migrations against a new database and against a reopened existing one.
- If concurrent branches repeatedly collide, adopt timestamp versions and
  document the change before enabling out-of-order migrations.

## 13. Configuration and secrets

- All process configuration is loaded through `internal/platform/config`.
- Validate configuration before opening external resources.
- Use explicit defaults only when they are safe for local development and
  production behavior remains obvious.
- Never commit secrets, credentials, real tokens, or production DSNs.
- Do not read environment variables throughout feature code.
- Redact secret values in errors and logs.
- Add every new environment variable to `README.md` and configuration tests.

## 14. Security baseline

The template supplies only a baseline. Authentication, sessions, CSRF/CORS, and
application authorization remain project-specific.

For every project:

- treat all request data as untrusted;
- use secure, HTTP-only, same-site cookies for sessions;
- require CSRF protection for browser-authenticated state changes;
- use CORS only for known origins and methods;
- authorize every protected operation server-side;
- use constant-time comparison where secret equality is checked;
- store passwords with a current password hashing scheme, not a fast hash;
- rotate or invalidate sessions after authentication state changes;
- set request size and time limits appropriate to the endpoint;
- avoid reflecting untrusted HTML;
- use `html/template` or `templ` escaping rather than manual concatenation;
- define a project-specific CSP when serving browser UI;
- keep dependencies and base images patched;
- run `govulncheck` in CI.

Read `docs/SECURITY.md` before adding authentication or multi-tenancy.

## 15. Tests

Every behavior change requires tests at the cheapest useful level.

- Pure business rules: table-driven unit tests.
- Services with ports: stubs/fakes owned by the feature test.
- SQLite adapters: real in-memory or temporary-file SQLite.
- HTTP translation: `httptest` with the real service or a narrow controlled
  dependency.
- Composition: app-level tests proving wiring and middleware.
- Infrastructure: focused tests for configuration, connection policy,
  migrations, transactions, and middleware order.

Rules:

1. Test public behavior, not private implementation line by line.
2. Keep tests deterministic; inject time and randomness.
3. Use `t.TempDir()` for filesystem state.
4. Use `t.Cleanup` for resources.
5. Do not use sleeps for synchronization when a channel or context can prove the
   condition.
6. Include negative and boundary cases.
7. For constraints, prove both the accepted and rejected path.
8. Run `go test -race ./...` before merging.

See `docs/TESTING.md`.

## 16. Logging and observability

- Use `log/slog` for structured logs.
- Prefer stable keys such as `request_id`, `method`, `path`, `status`, and
  `duration`.
- Log operational context, not sensitive payloads.
- Do not use logs as a substitute for returned errors.
- Keep health endpoints cheap and dependency semantics clear:
  `/healthz` is liveness; `/readyz` checks readiness.
- Add metrics/tracing vendors in platform packages, not features, unless a
  feature emits a business event through an injected interface.

## 17. Naming and code style

- Follow standard Go naming and `gofmt`.
- Package names are short, lowercase, and singular when natural.
- Prefer domain verbs: `Subscribe`, `Book`, `Cancel`, `Activate`.
- Avoid vague types such as `Manager`, `Helper`, `Util`, or `Common`.
- Avoid stuttering: `subscriber.Service`, not `subscriber.SubscriberService`.
- Keep exported API surface small.
- Write comments for exported identifiers and non-obvious invariants, not for
  obvious syntax.
- Prefer early returns over deep nesting.
- Prefer simple loops and explicit code over clever generic helpers.
- Introduce generics only when multiple real call sites become simpler.
- Do not add a dependency when the standard library is clear and sufficient.

## 18. Frontend assets

- The template imposes no frontend framework.
- Preserve accessible HTML and keyboard behavior.
- Treat `tokens.css` as neutral design primitives, not a component library.
- Do not add Tailwind, a bundler, or a JavaScript framework without a project
  requirement.
- Keep generated assets out of source directories and document the build step.
- Do not inline untrusted data into script contexts.

## 19. Docker and operations

- Preserve the non-root runtime user.
- Keep the runtime image minimal and the build reproducible.
- Do not bake secrets into images.
- Persist SQLite data on a mounted volume.
- Keep health checks aligned with the application's liveness contract.
- Ensure graceful shutdown remains functional.
- Back up the SQLite file using a method safe for a live WAL database; do not
  assume copying only the main file is always sufficient.

## 20. Prohibited changes without explicit architectural approval

Do not:

- create global `controllers/`, `models/`, `services/`, `repositories/`, or
  `ports/` directories;
- split every feature into `domain/application/adapters` subpackages;
- introduce a DI framework;
- introduce an ORM by default;
- replace `net/http` merely for routing convenience;
- make features register themselves through global state;
- move feature SQL into a central repository package;
- introduce generic CRUD repositories;
- add authentication, queues, caching, payments, or observability vendors to
  the base template without a concrete template-wide requirement;
- weaken tests, timeouts, security headers, non-root containers, or race checks
  merely to make a change pass.

## 21. Completion checklist for agents

Before declaring a task complete:

- [ ] The code has one obvious feature owner.
- [ ] Dependency direction still follows this document.
- [ ] Business rules are not hidden in HTTP or SQL adapters.
- [ ] New dependencies are justified.
- [ ] New configuration is validated and documented.
- [ ] New schema changes have a migration.
- [ ] Errors are classified and translated at the proper boundary.
- [ ] Security and tenant scoping were considered.
- [ ] Tests cover success, failure, and relevant boundary cases.
- [ ] `gofmt` has been run.
- [ ] `go build ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `go test -race ./...` passes.
- [ ] `golangci-lint run` passes when available.
- [ ] Documentation reflects architectural or operational changes.

- [`docs/AUTHENTICATION.md`](docs/AUTHENTICATION.md) — authentication flows and production boundaries.
