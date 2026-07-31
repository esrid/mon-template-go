# Architecture Decisions

This is a lightweight decision log. It records current defaults and their
tradeoffs so future changes can challenge an explicit choice rather than repeat
old debates.

## ADR-001 — Feature-first modular monolith

**Status:** Accepted

### Decision

Organize product code into vertical feature packages under
`internal/feature/<name>`.

### Rationale

A feature can be read, tested, changed, or removed from one directory. This
optimizes developer navigation and keeps business vocabulary visible.

### Rejected alternatives

- global `handlers/`, `services/`, `repositories/`, `models/` directories;
- a strict root-level `domain/application/ports/adapters` tree;
- microservices for the base template.

### Consequences

Feature packages contain several technical concerns in separate files but one Go
package. Cross-feature collaboration must be injected through consumer-owned
interfaces.

---

## ADR-002 — Ports live with consumers

**Status:** Accepted

### Decision

Declare dependency interfaces inside the feature that consumes them.

### Rationale

Consumers define the smallest useful contract. A central ports package tends to
be provider-shaped, accumulates unrelated interfaces, and couples features to a
shared abstraction catalogue.

### Consequences

A concrete adapter may satisfy several small interfaces structurally. Similar
interfaces may exist in different features; that duplication is acceptable when
the consumers have different needs.

---

## ADR-003 — Features do not import features

**Status:** Accepted

### Decision

Feature packages never import one another. They collaborate through interfaces
wired in `internal/app`.

### Rationale

This prevents cycles, preserves ownership, and keeps the composition root as the
only full-system view.

### Consequences

Dashboard/aggregation features may declare read interfaces and receive other
services. Small adapter methods may occasionally be needed at composition time.

---

## ADR-004 — One package per feature by default

**Status:** Accepted

### Decision

Keep each feature in one Go package and split by files before subpackages.

### Rationale

Premature `domain`, `application`, and `adapter` subpackages force internal types
to be exported and increase navigation without reducing real complexity.

### Consequences

Large features require discipline in file naming. A package may be split later
when clear boundaries and repeated pain justify it.

---

## ADR-005 — Standard-library HTTP server and router

**Status:** Accepted

### Decision

Use `net/http`, `http.Server`, and method-aware `http.ServeMux` patterns.

### Rationale

The standard library provides routing, middleware composition, server limits,
and lifecycle primitives required by this template without framework lock-in.

### Consequences

The project writes small response and middleware helpers itself. A framework may
be adopted by a concrete product only when it solves demonstrated requirements.

---

## ADR-006 — Manual dependency injection

**Status:** Accepted

### Decision

Construct and inject dependencies explicitly in `internal/app`.

### Rationale

The graph is small, compile-time visible, searchable, and requires no container
or reflection.

### Consequences

`app.go` grows one construction line/field per feature. If construction becomes
large, extract explicit `wireFeature` functions, not a service locator.

---

## ADR-007 — SQLite with WAL as the default database

**Status:** Accepted for the template

### Decision

Use `modernc.org/sqlite` through `database/sql`, with WAL/foreign keys/busy
timeout and a small connection policy.

### Rationale

SQLite gives a zero-service local and production deployment for many small web
applications. The pure-Go driver supports `CGO_ENABLED=0` production builds.

### Consequences

SQLite's write concurrency and operational model must fit the product. Projects
with demonstrated PostgreSQL requirements should add PostgreSQL-specific
adapters and migrations rather than pretending SQL dialects are identical.

---

## ADR-008 — Feature SQL local, migrations global

**Status:** Accepted

### Decision

Keep feature queries in the feature's `sqlite.go`, but keep all Goose migrations
in one globally ordered directory under `internal/platform/sqlite/migrations`.

### Rationale

Queries belong to the capability they implement. Goose, unlike Django's
migration system, maintains one sequence and does not resolve a dependency graph
across feature directories.

### Consequences

Copying a feature requires copying/creating its migration separately. Feature
integration tests must fail loudly when schema support is missing.

---

## ADR-009 — Business rules do not live in SQL adapters

**Status:** Accepted

### Decision

Keep algorithms and policy that can execute without a database in feature
business code.

### Rationale

Rules hidden in a storage adapter become harder to test, reuse from multiple
entrypoints, and recognize as authoritative.

### Consequences

Adapters may enforce atomicity and use database constraints, but domain decisions
such as transitions, capacity, limits, and idempotency semantics remain outside
SQL mechanics.

---

## ADR-010 — Inject time-dependent behavior

**Status:** Accepted

### Decision

Inject clocks/generators when a feature's result depends on ambient time or
randomness.

### Rationale

This keeps business tests deterministic and avoids global monkey-patching.

### Consequences

Constructors may accept small function types such as `func() time.Time`.

---

## ADR-011 — Production container runs as non-root

**Status:** Accepted

### Decision

Use a multi-stage build and run the final process as an unprivileged `app` user
with a dedicated writable `/data` directory.

### Rationale

This reduces image size and limits the impact of process compromise.

### Consequences

Writable paths and mounted volumes must have correct ownership. New runtime
features cannot assume root permissions or arbitrary filesystem writes.

---

## ADR-012 — Project-specific capabilities stay out of the base template

**Status:** Accepted

### Decision

Do not preinstall queues, caching, a concrete email provider,
payments, storage, or observability vendors.

### Rationale

These choices vary substantially by product and can impose security or vendor
assumptions. Empty scaffolding increases maintenance without proving value.

### Consequences

Each product adds the capability through feature-owned ports and platform
implementations, and documents its own decision.

---

## ADR-013 — Race tests and vulnerability checks are quality gates

**Status:** Accepted

### Decision

CI runs build, vet, `go test -race`, `govulncheck`, and golangci-lint.

### Rationale

The template should detect common correctness and dependency risks before a
project adds complexity.

### Consequences

Changes should fix race/lint/vulnerability findings rather than disable the
checks without a documented reason.

---

## Proposing a new decision

Add a new ADR section when a change affects several features or future projects.
Include:

- status;
- decision;
- context/rationale;
- alternatives rejected;
- consequences and migration impact.

Do not record every local implementation detail. Record choices future maintainers
might reasonably question.
