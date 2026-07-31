# Project Structure

This document is a directory map and placement guide.

## Repository tree

```text
.
├── cmd/
│   └── web/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   ├── routes.go
│   │   └── app_test.go
│   ├── feature/
│   │   ├── readiness/
│   │   └── subscriber/
│   └── platform/
│       ├── config/
│       ├── sqlite/
│       │   └── migrations/
│       └── web/
├── assets/
│   ├── src/
│   ├── dist/
│   └── templates/
├── docs/
├── .github/workflows/
├── Dockerfile
├── compose.yml
├── Makefile
├── go.mod
└── README.md
```

## `cmd/web`

### Purpose

Executable boundary for the HTTP process.

### Put here

- `main`;
- signal context setup;
- final fatal-error logging and exit.

### Do not put here

- route definitions;
- database queries;
- feature construction details;
- business rules;
- environment parsing beyond calling the config package.

## `internal/app`

### Purpose

Application composition and lifecycle.

### `app.go`

Owns concrete construction:

- configuration validation;
- opening shared infrastructure;
- building services;
- injecting dependencies;
- creating the HTTP server;
- graceful shutdown;
- closing resources.

### `routes.go`

Owns the list of mounted features and the final shared middleware stack.

### Do not put here

- form or JSON parsing;
- SQL;
- feature-specific status mapping;
- domain validation;
- reusable infrastructure helpers.

## `internal/feature`

### Purpose

Business capabilities organized as vertical slices.

Each direct child is a Go package representing one feature.

### `readiness`

Minimal infrastructure-facing slice used for liveness/readiness semantics. It
shows a service consuming a very narrow `Ping` port and two HTTP routes.

### `subscriber`

Full reference slice. Copy its shape when starting a typical persisted feature.
It demonstrates:

- business validation;
- feature errors;
- an injected clock;
- feature-owned ports;
- HTTP error mapping;
- feature-local SQLite queries;
- unit, HTTP, and integration tests.

### Naming new feature directories

Use the business noun understood by the product:

```text
appointment
customer
conversation
followup
identity
billing
```

Avoid technical or vague names:

```text
services
handlers
manager
utils
common
misc
```

## Files inside a feature

These names are conventions, not mandatory scaffolding.

### `<feature>.go`

Use cases, business types, validation, and feature error values.

### `port.go`

Interfaces and function types consumed by the feature. Keep the contracts narrow.

### `http.go`

Route mounting, request parsing, response serialization, and mapping feature
errors to HTTP status codes.

### `sqlite.go`

SQLite implementation of the feature's persistence port. Feature SQL remains
here rather than in a global repository directory.

### `<concern>.go`

When a feature grows, split files by meaningful concern while keeping one
package:

```text
appointment/
  appointment.go
  scheduling.go
  transition.go
  port.go
  http.go
  voice.go
  sqlite.go
```

Do not create subpackages merely to imitate layers.

## `internal/platform`

### Purpose

Shared technical capabilities with no feature-specific policy.

### `config`

Environment loading, defaults, parsing, validation.

A new variable requires:

- a `Config` field;
- loading/parsing logic;
- validation where appropriate;
- tests;
- README documentation.

### `sqlite`

Shared database mechanics:

- database opening;
- pragmas;
- connection limits;
- embedded Goose migrations;
- transactions;
- generic SQLite error inspection.

Feature queries do not belong here.

### `sqlite/migrations`

Single globally ordered schema history. Name each migration after the capability
or schema change it introduces.

### `web`

Generic middleware and response primitives. It may know HTTP concepts, but not
subscriber, billing, customer, or other business concepts.

Potential additions that fit here:

```text
web/session.go       only after a project selects a session design
web/csrf.go          project-specific browser policy
web/pagination.go    generic parsing/links if used by several features
```

Do not add speculative helpers.

## `assets`

### `assets/src`

Human-authored frontend sources.

### `assets/dist`

Generated or distributable assets. Keep its production process documented when
a build tool is introduced.

### `assets/templates`

Optional template files. The base repository does not impose `html/template`,
`templ`, or a frontend framework.

## `docs`

Engineering rules and rationale:

- `ARCHITECTURE.md`: system shape and dependency model;
- `PROJECT_STRUCTURE.md`: placement map;
- `STYLE_GUIDE.md`: code and naming conventions;
- `SECURITY.md`: security baseline and project additions;
- `TESTING.md`: test strategy;
- `CODE_REVIEW.md`: merge checklist;
- `DECISIONS.md`: architectural decision log.

`AGENTS.md` remains at the repository root so coding agents discover it easily.

## Root operational files

### `Makefile`

Stable developer commands. Keep targets small and composable.

### `Dockerfile`

Multi-stage production build, non-root runtime, persistent `/data` volume, and
liveness health check.

### `compose.yml`

Local container execution with a named SQLite data volume.

### `.github/workflows/ci.yml`

Build, vet, race tests, vulnerability analysis, and lint.

### `.golangci.yml`

Repository lint policy. Prefer a small high-signal policy over dozens of noisy
rules.

## Placement decision table

| Change | Location |
|---|---|
| New appointment validation | `internal/feature/appointment/` |
| Appointment SQL query | `internal/feature/appointment/sqlite.go` |
| Shared SQLite pragma | `internal/platform/sqlite/` |
| New migration | `internal/platform/sqlite/migrations/` |
| New feature route | that feature's `http.go` |
| Mounting a new feature | `internal/app/routes.go` |
| Constructing its service | `internal/app/app.go` |
| Environment variable | `internal/platform/config/` + README |
| Generic request middleware | `internal/platform/web/` |
| Feature-specific authorization rule | owning feature service |
| Process signal handling | `cmd/web/` or `internal/app` lifecycle |

## A simple placement test

Ask three questions:

1. Is this a product/business decision? Put it in the owning feature.
2. Is this technical machinery reused by multiple features? Put it in platform.
3. Is this construction or process lifecycle? Put it in app/cmd.

When none fits, reconsider whether the abstraction is necessary.
