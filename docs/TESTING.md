# Testing Guide

## Goals

Tests should make feature changes safe without coupling the suite to internal
implementation details.

The repository uses four complementary levels:

1. business unit tests;
2. adapter integration tests;
3. HTTP tests;
4. composition/infrastructure tests.

Use the cheapest level that proves the behavior, then add boundary coverage where
translation or integration can fail.

## Commands

```sh
make test       # go test -race ./...
make vet
make lint
make vuln
```

Useful focused commands:

```sh
go test ./internal/feature/subscriber
go test -run TestName ./internal/feature/subscriber
go test -race -count=1 ./...
go test -cover ./...
```

Do not rely only on cached test results while diagnosing concurrency or stateful
failures.

## Business tests

Test feature services with small stubs or fakes that implement feature-owned
ports.

Good targets:

- validation boundaries;
- normalization;
- idempotency behavior;
- state transitions;
- duplicate handling;
- time-dependent results;
- propagation/classification of dependency failures.

Inject clocks and generators so expected results are exact.

Prefer table-driven tests when cases share one behavior shape:

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name  string
        input string
        want  error
    }{
        // ...
    }
}
```

Do not create a generalized mocking framework. Feature-local stubs keep contracts
visible.

## SQLite integration tests

Use a real SQLite database for adapter behavior. Mocking `database/sql` does not
prove SQL syntax, migrations, constraints, scan order, or transaction semantics.

Choose database mode deliberately:

- `:memory:` for isolated fast tests where one connection is sufficient;
- a file under `t.TempDir()` for reopen, WAL, persistence, and connection-policy
  behavior.

Tests should cover:

- successful inserts/reads;
- uniqueness and foreign-key constraints;
- not-found behavior;
- transaction commit/rollback;
- migrations on a clean database;
- reopening an already migrated database;
- SQLite error translation.

Clean resources with `t.Cleanup`.

## HTTP tests

Use `net/http/httptest`.

Test the feature's transport responsibilities:

- method/path registration;
- valid parsing;
- malformed payloads;
- feature-error to status mapping;
- response content type and body shape;
- request context propagation;
- security/cache headers where the feature sets them.

Do not duplicate every service rule at the HTTP layer. One or two representative
mapping cases plus thorough service tests are usually better.

Use the real service with controlled ports when practical. This catches the
actual relationship between parsing, service behavior, and response mapping.

## App and route tests

`internal/app` tests prove what unit tests cannot:

- concrete infrastructure opens;
- features are constructed;
- routes are mounted;
- shared middleware wraps them;
- unknown routes remain 404;
- server configuration uses validated values.

Keep these tests broad but few. Feature behavior belongs in feature tests.

## Middleware tests

Middleware order is behavior. Test consequences rather than private function
ordering.

Examples already present:

- a panic becomes a 500;
- the same request receives a request ID;
- a panicking request still produces an access log with status 500;
- baseline security headers are present.

When adding middleware, test ordering interactions such as authentication,
compression, tracing, panic recovery, and response recording.

## Configuration tests

For each environment variable, test:

- default;
- valid override;
- invalid syntax;
- invalid bounds;
- backward-compatible fallback if one is intentionally supported.

Tests should not depend on the developer's environment. Use `t.Setenv` and clear
relevant variables explicitly.

## Concurrency and race detection

`go test -race ./...` is the default test target.

Do not disable it because a test is inconvenient. Fix shared state, package
globals, unsynchronized stubs, or lifecycle races.

For concurrency tests:

- coordinate with channels or wait groups;
- avoid arbitrary sleeps;
- prove completion with contexts/timeouts;
- run repeated counts when diagnosing flakes.

## Error assertions

Use:

```go
errors.Is(err, ErrSomething)
errors.As(err, &target)
```

Avoid exact wrapped-error strings. Assert text only when the text is itself the
public behavior being tested.

## Test data

- Use minimal values that communicate the case.
- Keep sensitive/real user data out of fixtures.
- Generate IDs deterministically when their exact value matters.
- Prefer helper functions local to the test package over global fixture systems.
- Mark helpers with `t.Helper()`.

## Golden files and snapshots

Use golden files only when output is large and stable enough that line-by-line
assertions are worse. Review diffs carefully; never update snapshots blindly.

For JSON APIs, decoding into a typed or small test struct is often clearer than a
full raw snapshot.

## Coverage

Coverage is a diagnostic, not a target that justifies low-value tests.

Prioritize:

- business invariants;
- security and tenant boundaries;
- failure translation;
- migrations and constraints;
- concurrency;
- process lifecycle.

Do not test standard-library behavior or trivial getters merely to increase a
percentage.

## Multi-tenant test matrix

When tenancy is introduced, every tenant-owned capability should include tests
for:

- tenant A reads A's record;
- tenant A cannot read B's record;
- tenant A cannot update/delete B's record;
- duplicate constraints are scoped correctly;
- list/search/count operations do not leak B;
- IDs guessed from another tenant remain inaccessible;
- background jobs and caches preserve tenant scope.

These are release-blocking tests.

## Test review checklist

- [ ] The test proves externally meaningful behavior.
- [ ] It runs deterministically.
- [ ] It cleans resources.
- [ ] It does not depend on execution order.
- [ ] It checks relevant boundary/failure cases.
- [ ] It uses real SQLite where SQL behavior matters.
- [ ] It uses `errors.Is` / `errors.As` for classifications.
- [ ] It passes under `-race`.
- [ ] It remains readable without knowledge of private implementation details.
