### Security posture
Security is best practice and best effort: hygiene, not audit requirements. No
formal audit or compliance obligation exists unless this project adds one and
says so here. The controls the scaffold ships (distroless nonroot runtime,
digest pins, hash-pinned actions, secret and CVE scanning) are described in
[docs/operations/security.md](docs/operations/security.md).

### Unfinished Todos
- Replace `myapp` and `github.com/myorg/myapp` placeholders with real values via `./scripts/init-template.sh`
- Adjust `docs/operations/security.md` once the service has real integrations
- Confirm the "CVE triage manifest" section below reflects this service's real
  gates -- the init script rewrites the repository name, not the workflow facts
- Replace this CLAUDE.md's "Common Tasks", "Key Paths", and "Architecture" sections with project-specific content
- Date D-001 in `docs/DECISIONS.md` (the scaffold entry) with the instantiation date
- Record the first real benchmark's written acceptance target, then re-record
  `.github/benchmarks/baseline.txt` with `task bench-baseline` -- the template
  ships no benchmarks and therefore no baseline

## Common Tasks
- Build: `task build` (or `go build ./cmd/myapp/`)
- Test: `go test -race ./...` (`go test -short ./...` skips docker-dependent
  integration suites when docker is unavailable)
- Lint: `task lint`
- Everything: `task all` (fmt, vet, lint, test, build)
- Benchmarks: `task bench` (every benchmark, `-count 6`, no tests and no
  docker), `task bench-compare` (benchstat against the committed
  `.github/benchmarks/baseline.txt`), `task bench-baseline` (rewrite that
  baseline -- a moved number lands in the PR diff and has to be justified).
  Not part of `task all`.
- Run server locally: `./myapp serve` (port :8080, override with `MYAPP_LISTEN_ADDR`)
- Deploy: push to `main`; [build.yml](.github/workflows/build.yml) builds + pushes to the configured registry
- Security scans: [security.yml](.github/workflows/security.yml) runs Trivy filesystem, TruffleHog, and Zizmor on every PR (inlined, hash-pinned public scanners)

## Key Paths
| Path | Purpose |
|---|---|
| `cmd/myapp/` | CLI entry point |
| `internal/cli/` | Cobra root and subcommands |
| `internal/server/` | HTTP server: probes, graceful shutdown |
| `internal/version/` | Build version (commit SHA injected via ldflags) |
| `docs/DECISIONS.md` | Decision log (D-### entries) |
| `docs/migrations/` | Phased rollout plans with validation appendices (see its README) |
| `docs/development/` | Doctrine: testing.md, database.md; local dev notes |
| `docs/operations/` | Runbooks and oncall references |
| `.github/workflows/` | CI/CD: validate on PR, build+push on merge |

## Architecture
- **stdlib first**: `log/slog` for logging, `net/http` for HTTP, `errors`/`fmt.Errorf("%w")` for error wrapping. Only external dependency is cobra.
- **Context propagation**: public functions take `ctx context.Context` first. Server lifetime context is threaded through background goroutines so cancellation reaches running work.
- **Graceful shutdown**: server's shutdown context is intentionally detached from the cancelled parent so the drain has its own timeout budget. The detach is annotated with `//nolint:contextcheck`.
- **Container hardening**: multi-stage build with distroless `nonroot` base; both layers digest-pinned for reproducibility.

## Development Patterns

**IMPORTANT:** If development conflicts with these ideas, STOP and outline the
conflict to the user and ask for input.

- All development follows TDD. Tests are written before implementation;
  "write and see if it compiles" is not a development strategy here.
- Development is modular and extensible: clean architecture with long-term
  thinking. Features fall out of deliberate architecture decisions rather
  than being stapled on.
- The domain core is pure Go with no I/O -- unit and property tests only.
  I/O lives in adapters (store, parsers, HTTP), which get integration and
  golden-file tests. See [docs/development/testing.md](docs/development/testing.md).
- Purity is not enough on its own: pure code must also be *reachable* without
  I/O. Computation that is only callable through a type whose construction
  requires a database is untestable and unbenchmarkable in practice, however
  pure its body is. Split load from fold and export the fold -- the seam rule
  is in [docs/development/database.md](docs/development/database.md) under
  "Store shape".
- If a database is added, follow
  [docs/development/database.md](docs/development/database.md): sqlc for
  compile-time-checked queries, embedded goose chain, caged migrate role,
  testcontainers integration tests.

### Performance

- Pure domain code gets a benchmark. If benchmarking it requires standing up
  a database, that is an architecture problem, not a testing problem -- fix
  the seam, not the test.
- Every benchmark states a written acceptance target and its rationale in the
  doc comment. A benchmark with no target cannot fail, so it cannot regress.
- Targets are checked by humans and benchstat, never a time-based
  `b.Fatalf`. Shared runners are too noisy for a latency threshold; only
  correctness invariants may fail a benchmark.
- A performance claim needs a measurement, not a subtraction. "The rest must
  be X" is a hypothesis -- profile or benchmark it before scoping work
  against it, because a plausible subtraction can be directionally right and
  still name the wrong function.
- Benchmarks rank and detect regressions; production telemetry attributes a
  request. Do not divide one into the other: they were measured on different
  CPUs and the ratio is meaningless.
- Slow handlers emit per-phase timing at debug plus a bounded-cardinality
  histogram. The always-on signal is the histogram, because a GET's own
  access log is usually debug-level too. Keep metric labels to declared
  constants so cardinality is bounded by construction, and pin the emitted
  label set with a test.

## Conventions

- Decisions worth carrying forward go in
  [docs/DECISIONS.md](docs/DECISIONS.md) as D-### entries, recorded when
  made.
- Plans above moderate complexity go in `docs/migrations/` with creation
  date and state (notstarted | started | complete), including before/after
  validation queries recorded in an appendix tracker. See
  [docs/migrations/README.md](docs/migrations/README.md).
- Backlog is GitHub issues in this repo.
- No emojis in documentation.
- Security controls are documented in
  [docs/operations/security.md](docs/operations/security.md). When a feature
  adds an integration or a destructive path, that doc gains a short section
  for it, referencing the D-### entries and plan doc that shaped it.

## CVE triage manifest

Consumed by the `/cve-ci-triage` skill (user-level, `~/.claude/skills/`). The
skill is the method; this section is this repository's facts. A repo cut from
this template is onboarded by running `./scripts/init-template.sh`, which
rewrites the `Repository` field below along with every other placeholder. No
hand-editing is required here.

- **Repository:** github.com/myorg/myapp
- **Ecosystem:** go (`go.mod`)
- **Scan gates:**
  | Workflow name | Job | File | Trigger | Target | Severity | ignore-unfixed | Red signal |
  |---|---|---|---|---|---|---|---|
  | Security Scans | Trivy Repository Scan | `.github/workflows/security.yml` | PR to main, Monday 14:00 UTC, dispatch | filesystem (`.`) | CRITICAL,HIGH | true | failed run only -- no tracking issue is opened |
- **Ignore files:** `.trivyignore.yaml` (repo root), YAML form, wired through the
  `trivyignores` input on the scan step.
- **Pin locations:** `go.mod` (application dependencies). `build.yml` builds and
  pushes but runs no Trivy scan, so no image base or tool binary is CVE-gated.
- **Pre-authorized merges:** trivyignore entries, dependency bumps, and green
  Renovate PRs may be self-merged. NetworkPolicy, RBAC, and secrets are not.
- **Repo notes:**
  - There is no tracking issue and no `issues: write` permission. A failed
    scheduled run on main is the only red signal, so check
    `gh run list --workflow "Security Scans"` rather than `gh issue list`.
  - Scans run on `ubuntu-latest`. A repo cut from this template that adopts the
    ARC runners should change `runs-on` in `security.yml` and say so here.
  - The weekly Monday run is the one that catches newly published CVEs against
    unchanged code; a PR run only sees the PR's own tree.
