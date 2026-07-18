# mon-template-go

Go web project template: hexagonal layout, SQLite (WAL) + goose migrations, manual DI.

## Layout

- `cmd/` — entrypoint, calls `internal/di`
- `internal/di/` — dependency wiring (add services/handlers in `App`)
- `internal/core/` — domain, ports, services
- `internal/adapters/` — stores (SQLite + migrations), handlers
- `assets/` — frontend sources and build output

## Usage

```sh
make run    # DSN=app.db by default
make test
make lint
```

Migrations live in `internal/adapters/stores/migrations/` (goose format, embedded).
