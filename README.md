# repo_template_go

Opinionated template for Go services that ship as a container image. Stand up a new repo with a working build, multi-stage distroless Dockerfile, race-tested CI, golangci-lint v2 baseline, pre-commit, and a structured-logging Cobra CLI scaffold in a few minutes.

## What you get

| Concern | Choice |
|---|---|
| CLI framework | [cobra](https://github.com/spf13/cobra) (one external dependency) |
| Logging | stdlib `log/slog` with JSON / text handler selection via flag |
| HTTP server | stdlib `net/http`, graceful shutdown, `/healthz` + `/readyz` probes |
| Build orchestration | [Taskfile](https://taskfile.dev) |
| Container | Multi-stage build, `gcr.io/distroless/static-debian12:nonroot`, digest-pinned |
| Lint | [golangci-lint v2](https://golangci-lint.run) with errorlint, gosec, bodyclose, contextcheck, revive, and friends |
| Pre-commit | gitleaks, golangci-lint, gofmt, go-mod-tidy, shellcheck, JSON-schema checks for workflows and Taskfile |
| CI | GitHub Actions: PR validate (build / vet / test -race / lint), main build + push to ACR, defense-in-depth security scans (Trivy filesystem, TruffleHog, Zizmor) |
| Compliance scaffolding | `SECURITY.md` template aligned with HIPAA, SOC 2, ISO 27001, HITRUST |

## Use this template

### Option 1: GitHub "Use this template"

Click `Use this template` on the GitHub UI, then run the rename script in the new clone:

```bash
./scripts/init-template.sh github.com/yourorg/yourapp yourapp
```

### Option 2: clone manually

```bash
git clone https://github.com/veyloris/repo_template_go.git yourapp
cd yourapp
rm -rf .git && git init
./scripts/init-template.sh github.com/yourorg/yourapp yourapp
```

The script does a recursive find/replace of `github.com/myorg/myapp` and `myapp` across source, configs, docs, and workflows.

## Quickstart after rename

```bash
go mod tidy
task build           # produces ./yourapp
task test            # go test -race -coverprofile=coverage.out ./...
task lint            # golangci-lint v2
./yourapp version    # prints commit
./yourapp serve      # starts server on :8080 with /healthz + /readyz
```

## Layout

```
.
├── cmd/myapp/                  # main package; one file, calls cli.NewRootCmd
├── internal/
│   ├── cli/                    # Cobra root + subcommands (version, serve)
│   ├── server/                 # HTTP server: probes, graceful shutdown
│   └── version/                # build-version injection via ldflags
├── docs/
│   ├── development/            # local dev notes (port-forward recipes, etc.)
│   ├── migrations/             # phased rollout plans (per CLAUDE.md preference)
│   └── operations/             # runbooks, oncall references
├── scripts/init-template.sh    # rename helper
├── .github/workflows/          # validate (PR) + build (main) + security (PR + push)
├── .golangci.yml               # v2 baseline; tune per project
├── .pre-commit-config.yaml     # gitleaks + lint + format + filesystem checks
├── Dockerfile                  # multi-stage distroless, digest-pinned
├── Taskfile.yml                # build / test / lint / fmt / vet / tidy / all
├── SECURITY.md                 # compliance + controls (template)
└── CLAUDE.md                   # project-level Claude Code memory (template)
```

## Opinions baked in

- **`internal/` only, no `pkg/`.** This template assumes you are shipping an application, not a library. If you genuinely need to expose code to other modules, add a `pkg/` directory yourself.
- **stdlib first.** Logging is `log/slog`. HTTP is `net/http`. Errors are `fmt.Errorf("...: %w", err)`. The only direct dependency is cobra; everything else is the standard library. Resist the urge to pull in zap, logrus, gin, chi, gorilla, or pkg/errors.
- **Context everywhere.** Public functions take `ctx context.Context` as the first argument. The server's lifetime context is propagated to background work so cancellation reaches running goroutines.
- **Distroless + nonroot.** No shell in the runtime image. Set `runAsNonRoot: true` on the pod and add `securityContext.allowPrivilegeEscalation: false` in your manifests.
- **Pin everything.** Image base layers and all GitHub Actions are digest-pinned. Renovate (or your dependency tool of choice) bumps them with a paper trail.
- **Pre-commit gates locally what CI gates remotely.** Same lint version is used in both. The `language_version` pin in `.pre-commit-config.yaml` ensures the locally-built linter binary is built with the project's Go toolchain version, avoiding "linter built with go1.X cannot lint a go1.Y project" failures.

## What is intentionally not included

- Database layer, ORM, migrations tooling. Add when you have a real schema.
- Metrics / tracing libraries. Add `prometheus/client_golang` or OpenTelemetry once you know what to instrument.
- Config loading framework (viper, koanf). Use `os.Getenv` and a small struct; add a loader if it earns its keep.
- A `Makefile`. Use Taskfile.
- A `pkg/` directory. See above.

## Compliance posture

This template assumes the resulting service operates under HIPAA, SOC 2, ISO 27001, and HITRUST. `SECURITY.md` documents the default controls (no-shell runtime, digest-pinned base images, nonroot user, structured audit logs, secrets via env-injection from a secret manager). When you instantiate, edit `SECURITY.md` to reflect the actual data classification, integrations, and access patterns of your service.
