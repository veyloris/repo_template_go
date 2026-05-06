### Compliance
This repository is developed with **HIPAA**, **HITRUST**, **SOC 2**, and **ISO 27001**
security standards in mind. All changes must account for these requirements. All data
processed is assumed PHI. See [SECURITY.md](SECURITY.md) for controls and compliance mapping.

### Unfinished Todos
- Replace `myapp` and `github.com/myorg/myapp` placeholders with real values via `./scripts/init-template.sh`
- Edit `SECURITY.md` to reflect this service's actual data flow, classifications, and integrations
- Replace this CLAUDE.md's "Common Tasks", "Key Paths", and "Architecture" sections with project-specific content

## Common Tasks
- Build: `task build` (or `go build ./cmd/myapp/`)
- Test: `go test -race ./...`
- Lint: `task lint`
- Run server locally: `./myapp serve` (port :8080, override with `MYAPP_LISTEN_ADDR`)
- Deploy: push to `main`; [build.yml](.github/workflows/build.yml) builds + pushes to the configured registry
- Security scans: [security.yml](.github/workflows/security.yml) runs Trivy filesystem, TruffleHog, and Zizmor on every PR (org reusable workflows)

## Key Paths
| Path | Purpose |
|---|---|
| `cmd/myapp/` | CLI entry point |
| `internal/cli/` | Cobra root and subcommands |
| `internal/server/` | HTTP server: probes, graceful shutdown |
| `internal/version/` | Build version (commit SHA injected via ldflags) |
| `docs/migrations/` | Phased rollout plans and migration trackers |
| `docs/development/` | Local dev notes (port-forward recipes, env setup) |
| `docs/operations/` | Runbooks and oncall references |
| `.github/workflows/` | CI/CD: validate on PR, build+push on merge |

## Architecture
- **stdlib first**: `log/slog` for logging, `net/http` for HTTP, `errors`/`fmt.Errorf("%w")` for error wrapping. Only external dependency is cobra.
- **Context propagation**: public functions take `ctx context.Context` first. Server lifetime context is threaded through background goroutines so cancellation reaches running work.
- **Graceful shutdown**: server's shutdown context is intentionally detached from the cancelled parent so the drain has its own timeout budget. The detach is annotated with `//nolint:contextcheck`.
- **Container hardening**: multi-stage build with distroless `nonroot` base; both layers digest-pinned for reproducibility.

## Documentation
- Phased rollout plans, migration trackers, and design docs go in `docs/migrations/`.
- Compliance and security controls documented in [SECURITY.md](SECURITY.md).
