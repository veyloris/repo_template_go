# Testing doctrine

Tests come before implementation. "Write it and see if it compiles" is not a
development strategy; the compiler proves types, tests prove behavior. Every
bug fix starts with a failing test that reproduces it.

## The layer model

Structure the module so each layer gets the cheapest test that actually
proves it:

| Layer | Contents | Test style |
|---|---|---|
| Domain core | Pure Go, no I/O imports | Unit + property tests, exhaustive |
| Adapters | DB store, parsers, HTTP clients | Integration (testcontainers), golden files |
| Server | Handlers over narrow interfaces | Hand-written fakes + `httptest` |
| End-to-end | Container -> migrate -> store -> server | One suite driving real HTTP |

The domain core (business rules, calculations, state machines) is pure
functions over plain data. I/O lives in adapters. The server defines its own
narrow consumer interfaces (`OrderStore`, `RuleStore`, ...) rather than
importing the persistence concretions; compile-time conformance assertions
keep them honest:

```go
var _ server.OrderStore = (*store.Store)(nil)
```

## Organization

- Tests are colocated: `store/orders.go` -> `store/orders_test.go`. No
  central `tests/` tree.
- External test packages (`package store_test`) are the default. White-box
  tests are the exception, named for intent
  (`sources_internal_test.go`, `package store`), used only when the test must
  reach unexported functions.
- Table-driven tests use inline anonymous structs iterated directly. Subtests
  (`t.Run`) group phases that share one expensive fixture (for example, one
  container serving several lifecycle scenarios).
- Helpers take `(ctx, t, ...)`, call `t.Helper()`, and register cleanup with
  `t.Cleanup`. No `TestMain`; each test provisions what it needs.
- Assertions are stdlib `t.Errorf`/`t.Fatalf`. No mock frameworks: doubles
  are hand-written fakes implementing the narrow server interfaces.
- The race detector is always on: `go test -race ./...` locally and in CI.

## Integration tests: testcontainers + `-short`

Docker-dependent suites are guarded by `testing.Short()`, so
`go test -short ./...` is the docker-free subset. There is no runtime docker
probing -- the flag is the contract, and CI runs both subsets (see
validate.yml: a `-short` job plus a full-suite job on a docker-capable
runner).

The canonical opening block:

```go
if testing.Short() {
    t.Skip("integration test: requires docker")
}
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
defer cancel()
pgc, err := tcpostgres.Run(ctx, "postgres:18-alpine",
    tcpostgres.WithDatabase("myapp"),
    tcpostgres.WithUsername("postgres"),
    tcpostgres.WithPassword("postgres"),
    tcpostgres.WithInitScripts("../db/testdata/bootstrap.sql"),
    tcpostgres.BasicWaitStrategies())
// handle err
t.Cleanup(func() { _ = pgc.Terminate(context.Background()) })
```

`bootstrap.sql` recreates the production role model, not a superuser
shortcut: the caged migrate role, the runtime role, and the
`ALTER DEFAULT PRIVILEGES` chain between them (see
[database.md](database.md)). Tests migrate as the migrate role and operate as
the runtime role, so grant and default-privilege regressions fail in CI
instead of in production.

## Migration tests: `MigrateUpTo -> seed -> MigrateUp`

A migration that transforms existing rows must be tested against pre-existing
rows. An empty table exercises nothing: a new CHECK constraint or backfill
that passes CI on empty tables can still reject or mangle real rows in
production. Expose a staging helper next to `MigrateUp`:

```go
// MigrateUpTo applies pending migrations up to and including version. Used to
// stage the chain at a point where pre-existing data can be seeded before a
// later migration runs, so data-migration paths are testable (an empty table
// never exercises them).
func MigrateUpTo(ctx context.Context, conn *sql.DB, version int64) error
```

Every data-transforming migration gets a test shaped:

1. `MigrateUpTo(ctx, conn, N-1)` -- stage the chain just before it.
2. Seed rows under the old schema with raw INSERTs as the migrate role.
3. `MigrateUp(ctx, conn)` -- the migration must succeed over those rows and
   transform them.
4. Assert the transformed state, and that any new constraint is actually in
   force (attempt a violating write and require the exact SQLSTATE).

Assert Postgres errors by SQLSTATE via `*pgconn.PgError`, not string match:

```go
const pgErrCheckViolation = "23514"       // requireCheckViolation
const pgErrInsufficientPrivilege = "42501" // requirePermissionDenied
```

## Golden-file tests: parsers

File-format parsers (PDF, CSV, API payloads) are tested against `testdata/`
fixtures with committed golden outputs:

```go
var update = flag.Bool("update", false, "update golden files")
// in the test:
gotJSON, _ := json.MarshalIndent(got, "", "  ")
if *update {
    os.WriteFile(goldenPath, append(gotJSON, '\n'), 0o644)
    return
}
want, err := os.ReadFile(goldenPath) // error text: "run with -update to create"
```

Fixture rules:

- Fixtures derived from real inputs are scrubbed of PII/PHI before commit,
  say so in the filename (`order-SCRUBBED-....golden.json`), and carry an
  in-file note distinguishing fake identity from real data shapes.
- For binary formats, fix the extraction stage's output (for a PDF, the page
  text) rather than the binary, so the golden pins your parser, not the
  third-party extractor.

## Differential tests: porting an existing system

When reimplementing something that already runs (a SQL view, a legacy
calculation), the acceptance gate is a differential harness, not spot checks:
load the same input into both implementations and compare row-for-row,
column-by-column. Compare fixed-point columns exactly and floats within a
relative epsilon (`1e-9 * max(1, |a|, |b|)`). Copy the reference
implementation verbatim into `testdata/` so the harness is self-contained,
and keep an env-gated variant that runs the same comparator against real
production-shaped data (skipped unless the DSN env var is set; never in CI).

Multi-statement seed scripts need a simple-protocol connection:
`dsn + "&default_query_exec_mode=simple_protocol"` with pgx.

## Property tests

Pure parsers and codecs get `testing/quick` property tests alongside
example-based tables -- round-trip fixed points are the workhorse:

```go
// Parse(p.Canonical()) must deep-equal p.
quick.Check(fn, &quick.Config{MaxCount: 2000})
```

## Benchmarks

Benchmarks use the Go 1.24+ `for b.Loop()` form, are sized to production
scale, and state their acceptance target in a comment ("full recompute well
under 1s at 54k SKUs") so a regression is a red number, not a shrug.
