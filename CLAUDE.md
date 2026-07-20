### Compliance
This repository is developed with **HIPAA**, **HITRUST**, **SOC 2**, and **ISO 27001**
security standards in mind. All changes must account for these requirements. All data
processed is assumed PHI. See [SECURITY.md](SECURITY.md) for controls and compliance mapping.

### Unfinished Todos
- Replace `myapp` and `github.com/myorg/myapp` placeholders with real values via `./scripts/init-template.sh`
- Edit `SECURITY.md` to reflect this service's actual data flow, classifications, and integrations
- Replace this CLAUDE.md's "Common Tasks", "Key Paths", and "Architecture" sections with project-specific content
- Date D-001 in `docs/DECISIONS.md` (the scaffold entry) with the instantiation date

## Common Tasks
- Build: `task build` (or `go build ./cmd/myapp/`)
- Test: `go test -race ./...` (`go test -short ./...` skips docker-dependent
  integration suites when docker is unavailable)
- Lint: `task lint`
- Everything: `task all` (fmt, vet, lint, test, build)
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
- If a database is added, follow
  [docs/development/database.md](docs/development/database.md): sqlc for
  compile-time-checked queries, embedded goose chain, caged migrate role,
  testcontainers integration tests.

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
- Compliance and security controls documented in [SECURITY.md](SECURITY.md).
  When a feature adds a data flow, SECURITY.md gains a controls subsection
  for it, referencing the D-### entries and plan doc that shaped it.
