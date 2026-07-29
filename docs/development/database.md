# Adding Postgres

The template ships without a database layer on purpose. When the service
grows a real schema, this is the proven recipe. It assumes the service may
share a cluster with other schemas it must not touch; nothing here hurts a
dedicated cluster.

## Roles: two DSNs, caged migrate role

Never run migrations as the runtime user, and never let the runtime user do
DDL. Provision (in your infra repo, not here):

- `svc_myapp_migrate` -- owns the app schema, does DDL. Used only by
  `myapp migrate`, DSN from `MYAPP_MIGRATE_DATABASE_URL`.
- `svc_myapp` -- runtime role, DML only on app tables, read-only on any
  external schemas. DSN from `MYAPP_DATABASE_URL`.
- `ALTER DEFAULT PRIVILEGES FOR ROLE svc_myapp_migrate IN SCHEMA myapp
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO svc_myapp;` -- so new
  tables are usable without per-migration grant boilerplate.

The integration-test bootstrap script (`internal/db/testdata/bootstrap.sql`)
recreates this exact role model so tests migrate as the migrate role and run
as the runtime role; a missing grant then fails in CI, not production. See
[testing.md](testing.md).

## Schema ownership boundary

- The app owns exactly one schema and its migration chain. Schema-qualify
  everything in migrations and queries (`myapp.orders`, never `orders`) --
  do not depend on `search_path`.
- Other schemas in the cluster are external contracts: read-only, enforced by
  grants in both directions.
- Enforce the boundary twice: a role-cage integration test (migrate role
  cannot touch external schemas, runtime role can read but not write them,
  neither can the other's job), and a static test that walks every non-test
  `.go` file for schema-qualified identifiers and fails on any external
  relation not declared in a reviewed source registry. New cross-boundary
  reads then require a registry entry, which makes them visible in review.

## Migrations: goose, embedded, one chain

`internal/db/migrate.go`:

```go
//go:embed migrations/*.sql
var embedded embed.FS

func newProvider(conn *sql.DB) (*goose.Provider, error) {
    fsys, _ := fs.Sub(embedded, "migrations")
    return goose.NewProvider(goose.DialectPostgres, conn, fsys,
        goose.WithTableName("myapp.goose_db_version")) // schema-qualified
}

func Open(ctx context.Context, dsn string) (*sql.DB, error)   // pgx stdlib driver + ping
func MigrateUp(ctx context.Context, conn *sql.DB) error
func MigrateUpTo(ctx context.Context, conn *sql.DB, version int64) error // test staging
func MigrateStatus(ctx context.Context, conn *sql.DB) ([]MigrationState, error)
```

- The embedded FS means the binary migrates itself: `myapp migrate up` and
  `myapp migrate status` are cobra subcommands reading
  `MYAPP_MIGRATE_DATABASE_URL`. No goose CLI on the host, no drift between
  deployed code and applied schema.
- `MigrateUpTo` exists for tests only: stage the chain, seed old-schema rows,
  then `MigrateUp` over them ([testing.md](testing.md), migration tests).
- Migrations that transform data are written expecting non-empty tables.

## Queries: sqlc, generated code checked in

`sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    # The prelude declares what exists outside the goose chain (the schema
    # itself, external read-only schemas); sqlc understands goose files and
    # ignores their Down sections.
    schema:
      - "internal/store/schema_prelude.sql"
      - "internal/db/migrations"
    queries: "internal/store/queries"
    gen:
      go:
        package: "storegen"
        out: "internal/store/storegen"
        sql_package: "pgx/v5"
        emit_json_tags: true
        overrides:
          - db_type: "pg_catalog.numeric"
            go_type: "float64"
          - db_type: "pg_catalog.numeric"
            nullable: true
            go_type: { type: "float64", pointer: true }
          - db_type: "pg_catalog.date"
            go_type: "time.Time"
          - db_type: "pg_catalog.timestamptz"
            go_type: "time.Time"
          - db_type: "pg_catalog.timestamptz"
            nullable: true
            go_type: { type: "time.Time", pointer: true }
```

- Pointing sqlc's `schema` at the goose chain means every query is
  compile-time checked against the real migrations; the prelude
  (`CREATE SCHEMA myapp;` plus external-schema stubs) covers objects the
  chain does not create.
- Regenerate with a pinned Taskfile target
  (`go run github.com/sqlc-dev/sqlc/cmd/sqlc@vX.Y.Z generate`); never
  hand-edit `storegen/`. Exclude the generated dir from most linters in
  `.golangci.yml`.
- Add a CI drift check so a stale regeneration cannot merge:
  `task sqlc && git diff --exit-code internal/store/storegen`.
- Raw SQL outside sqlc is an exemption to justify per query (dynamic filter
  grammars, LATERAL-heavy search), not an alternative style.

## Store shape

- `internal/store/queries/*.sql` hand-written, one file per concern;
  `internal/store/storegen/` generated; domain-facing store types embed a
  shared `base{pool *pgxpool.Pool; q *storegen.Queries}` with a single
  `withTx` transaction frame.
- Store behavior is integration-tested against real Postgres via
  testcontainers. Never mock sqlc: the generated layer's value is that it is
  compile-time checked, and mocking it tests the mock.

That shape is right for I/O. It is wrong for computation.

**Pure computation does not live on a store receiver.** `base` has no
exported constructor, so the only way to obtain a receiver is to dial
Postgres -- which means anything reachable only through it needs a live
database to exercise. A fold that is genuinely pure but unreachable without
Postgres will not get benchmarked, and in practice does not: standing up a
container to time in-memory arithmetic is enough friction that nobody pays
it, and the hot path ends up the one part of the domain core with no numbers
on it.

When a method loads rows and then folds them, split it in three:

```go
// I/O only. One query set, no computation.
func (s *Store) loadWidgetInputs(ctx context.Context, since time.Time) (WidgetInputs, error) {
    rows, err := s.q.WidgetsSince(ctx, since)
    if err != nil {
        return WidgetInputs{}, fmt.Errorf("load widget inputs: %w", err)
    }
    return WidgetInputs{Widgets: rows}, nil
}

// Pure. No receiver, no ctx, no pool -- exported so a benchmark can call it.
func WidgetSummaryFold(in WidgetInputs) Summary { ... }

// Thin delegate. The original method, so no caller changes.
func (s *Store) WidgetSummary(ctx context.Context, since time.Time) (Summary, error) {
    in, err := s.loadWidgetInputs(ctx, since)
    if err != nil {
        return Summary{}, err
    }
    return WidgetSummaryFold(in), nil
}
```

Two things fall out of the split for free. It is the natural seam for
timing, so query time and compute time are two numbers instead of one opaque
one. And a synthetic fixture can drive the fold directly with no database,
which is what makes a scale-parameterised benchmark possible at all --
see [testing.md](testing.md).

## Dependencies

`github.com/jackc/pgx/v5` (driver + pool), `github.com/pressly/goose/v3`
(migrations), `github.com/testcontainers/testcontainers-go/modules/postgres`
(tests). Record the adoption and any deviation from this recipe as a D-###
entry in [docs/DECISIONS.md](../DECISIONS.md).
