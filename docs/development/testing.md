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
scale, and state their acceptance target in the doc comment so a regression
is a red number, not a shrug.

**A benchmark with no written target cannot fail, so it cannot regress.** The
target and the derivation that produced it belong in the doc comment, not in
whoever wrote it:

```go
// BenchmarkWidgetSummaryFold folds one full reporting window in memory.
//
// Target: under 250ms at 200k widgets. The handler that calls it has a 2s
// budget and spends ~1.2s in queries, so the fold has to stay well inside
// the remainder or the page is the fold.
func BenchmarkWidgetSummaryFold(b *testing.B) { ... }
```

**Targets are checked by humans and `benchstat`, never a time-based
`b.Fatalf`.** Shared CI runners are far too noisy for a latency threshold;
a wall-clock assertion there fails on neighbors, gets marked flaky, and then
gets deleted. Only correctness invariants may fail a benchmark -- and they
should: a conservation law or an expected output count also stops a
benchmark from silently measuring a fold that early-returned on degenerate
input.

**Benchmarks rank; telemetry attributes.** A benchmark is measured on the
developer's CPU and a request latency on the deployment's, and the same code
can be several times slower on a small throttled container. Dividing a
benchmark number into a production latency to get a "share of the request"
is a real and easy mistake that yields a figure wrong by an order of
magnitude. Use benchmarks to compare versions of the same code on the same
machine and to catch regressions; use production histograms to attribute
where a request's time actually goes.

**Fixture generators are synthetic, deterministic, and
scale-parameterised.** They live in their own package
(`internal/fixtures`), are seeded from a fixed seed so runs compare, and are
never seeded from a production dump or an anonymized extract -- a
git-tracked fixture carries none of the access controls the source rows sit
behind, and "anonymized" is a claim nobody re-checks after the first commit.
Make scale a parameter, so "at twice the data, is this twice the time or
four times?" is a benchmark rather than an opinion. Identity-shaped
generated values (names, emails, account numbers) should be obviously fake,
so a leaked fixture can never be mistaken for real data.

**Guard the fixture package's import graph with a test.** If the generators
import the store package they drag the database driver into the test binary
of every pure package that uses them, and the pure packages quietly stop
being cheap. Ask the toolchain rather than grepping, because the hazard is a
*transitive* import:

```go
func TestFixturesDoNotDependOnStore(t *testing.T) {
    out, err := exec.Command("go", "list", "-deps",
        "github.com/myorg/myapp/internal/fixtures").Output()
    if err != nil {
        t.Fatalf("go list -deps: %v", err)
    }
    for _, dep := range strings.Fields(string(out)) {
        if dep == "github.com/myorg/myapp/internal/store" ||
            strings.HasPrefix(dep, "github.com/jackc/pgx") {
            t.Errorf("fixtures must not depend on %s", dep)
        }
    }
}
```

A second `go list -deps ./cmd/myapp` test asserting `internal/fixtures` is
absent keeps a test-only generator from being linked into the shipped
binary.

**A degenerate corpus is worse than no benchmark.** Well-typed fixture data
that makes every fold hit its empty-input branch produces a fast number that
means nothing, and it will be quoted. Have the generator self-check the
invariants the code under test actually depends on -- non-empty groups,
matching join keys, values in the ranges the branches care about -- and fail
loudly when it produces a corpus that does not exercise them.

**`go test -bench` and containers.** `-run '^$'` selects no test, which is
what keeps testcontainers suites from starting docker during a benchmark
run. That holds only while no `TestMain` starts a container unconditionally;
if you add one, it must respect `testing.Short()`. Never combine `-race`
with benchmarks -- roughly 10x cost, and the resulting numbers compare to
nothing.

Run them with `task bench`, compare with `task bench-compare`, re-record
with `task bench-baseline`. The committed baseline lives in
[.github/benchmarks/](../../.github/benchmarks/).
