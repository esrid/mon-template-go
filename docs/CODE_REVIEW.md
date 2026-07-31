# Code Review Checklist

Use the sections relevant to the change. A checked box means the reviewer has
verified it, not merely that the author believes it is true.

## Ownership and architecture

- [ ] The change has one obvious feature owner.
- [ ] New business code lives under `internal/feature/<name>`.
- [ ] Shared technical code lives under `internal/platform` only when multiple
      features genuinely use it.
- [ ] `internal/app` contains construction/composition, not feature behavior.
- [ ] Feature packages do not import one another.
- [ ] Platform packages do not import feature packages.
- [ ] New interfaces are declared by their consumer and are narrow.
- [ ] No speculative global abstraction or generic CRUD repository was added.
- [ ] A small feature was not split into ceremonial layer subpackages.

## Business rules

- [ ] Authoritative validation and invariants live in feature business code.
- [ ] HTTP and SQLite layers only translate their boundary concerns.
- [ ] Business constants have one source of truth.
- [ ] State transitions, quotas, capacity, idempotency, and authorization rules
      are explicit and tested.
- [ ] Time/randomness is injected where deterministic behavior matters.

## HTTP

- [ ] The feature owns its complete method-aware route paths.
- [ ] No feature accidentally registers `/` as a fallback.
- [ ] Input size and content type are bounded/validated where needed.
- [ ] Request context is propagated.
- [ ] Feature errors map to intentional status codes.
- [ ] Internal errors and sensitive information are not returned.
- [ ] Response content type and cache behavior are explicit.
- [ ] State-changing browser endpoints have project-appropriate CSRF protection.
- [ ] Authentication and authorization are both enforced where required.

## SQL and migrations

- [ ] Queries use placeholders for untrusted values.
- [ ] Dynamic identifiers/order clauses are allowlisted.
- [ ] Queries select explicit columns and check all errors.
- [ ] Rows are closed and `Rows.Err()` is checked.
- [ ] Atomic invariants use a correctly scoped transaction.
- [ ] Database constraints back important uniqueness/reference rules.
- [ ] Driver errors are translated before escaping the adapter.
- [ ] Business algorithms were not buried in `sqlite.go`.
- [ ] A schema change has a new globally ordered migration.
- [ ] Existing applied migrations were not edited.
- [ ] Clean migration and reopen/upgrade paths are tested.

## Security and privacy

- [ ] Trust boundaries and permissions are understood.
- [ ] No secrets, tokens, cookies, auth headers, or unnecessary personal data are
      logged.
- [ ] Session cookies use secure attributes when sessions exist.
- [ ] CORS is absent or narrowly allowlisted.
- [ ] Browser output is escaped and CSP implications were considered.
- [ ] Webhooks verify signatures and replay/idempotency where applicable.
- [ ] File uploads are size-limited, renamed server-side, and authorized.
- [ ] Multi-tenant reads/writes/constraints/caches/jobs are tenant-scoped.
- [ ] Cross-tenant negative tests exist.
- [ ] New dependencies were reviewed for maintenance, license, and security.

## Configuration and operations

- [ ] New environment variables are loaded centrally.
- [ ] Invalid configuration fails before external resources open.
- [ ] Defaults are safe and documented.
- [ ] Secrets are not committed or baked into images.
- [ ] Graceful shutdown and resource closure remain correct.
- [ ] The container still runs as non-root.
- [ ] Health/readiness semantics remain accurate.
- [ ] SQLite persistence/backup implications are documented when changed.

## Tests

- [ ] Behavior changes have tests at the cheapest useful level.
- [ ] Business tests cover success, failure, and boundaries.
- [ ] SQL behavior uses real SQLite tests.
- [ ] HTTP translation uses `httptest`.
- [ ] Construction/routing changes have app-level coverage.
- [ ] Tests are deterministic and clean resources.
- [ ] Error classifications use `errors.Is` / `errors.As`.
- [ ] Concurrency changes pass the race detector.

## Quality gates

- [ ] `gofmt` has been applied.
- [ ] `go build ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `go test -race ./...` passes.
- [ ] `golangci-lint run` passes.
- [ ] `govulncheck ./...` has no reachable unresolved vulnerability.
- [ ] README/docs were updated for public behavior, config, architecture, or
      operational changes.

## Final reviewer question

Can a developer unfamiliar with the patch find the new behavior and understand
its dependencies without searching through multiple technical-layer folders?

When the answer is no, improve ownership before merging.
