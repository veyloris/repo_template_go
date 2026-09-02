# Security hygiene

This document describes the security controls the scaffold provides for
`myapp`. The posture is best practice and best effort: hygiene, not audit
requirements. Nothing here is a compliance attestation, and no formal audit
obligation exists unless the instantiating project adds one and says so.

Statements below describe what the scaffold ships. They become true of a
deployed service only once it is configured that way.

> Template note: replace `myapp` and adjust the sections that describe
> integrations once the service has real ones.

## Container hardening

- Multi-stage build; runtime image is `gcr.io/distroless/static-debian12:nonroot`
  with no shell or package manager.
- Static binary (`CGO_ENABLED=0`) so the runtime image needs no glibc.
- Runs as `nonroot` user (UID 65532).
- Both base images are pinned by digest.
- Recommended pod security context:
  ```yaml
  securityContext:
    runAsNonRoot: true
    runAsUser: 65532
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    seccompProfile:
      type: RuntimeDefault
    capabilities:
      drop: [ALL]
  ```

## Supply chain

GitHub Actions workflows are audited by zizmor on every PR (see
[security.yml](../../.github/workflows/security.yml)). The action pinning
policy is enforced through [.github/zizmor.yml](../../.github/zizmor.yml):

- All actions (e.g. `actions/checkout`, `golangci/golangci-lint-action`) must
  be pinned to a full commit SHA, with the tag recorded in a trailing comment.
  Renovate keeps these digests current.
- This template uses no org-internal reusable workflows. If your org adds some
  inside its own trust boundary, they may be referenced by ref (`@main`) so
  centrally managed security tooling propagates without per-repo pin bumps;
  add the corresponding `ref-pin` policy to `.github/zizmor.yml` and record
  the rationale here.

Dependency CVEs are gated by Trivy in the same workflow. The gate's facts
(severity, ignore file, pin locations) live in the "CVE triage manifest"
section of [CLAUDE.md](../../CLAUDE.md), which the `/cve-ci-triage` skill
consumes.

## Secrets

```
External secret manager (source of truth)
        |
        | ExternalSecrets / CSI driver
        v
K8s Secret (in the service's namespace)
        |
        | env var injection
        v
myapp container
```

- Secrets never appear in git, CI logs, or container images. gitleaks runs in
  pre-commit and TruffleHog in CI to catch the cases where that rule slips.
- Access to the secret manager is via workload identity federation, not
  static credentials, where the platform supports it.
- Secrets are scoped to the service's namespace and not readable
  cross-namespace.

## Network

- The service is reachable only via internal cluster networking unless an
  explicit ingress is configured.
- Egress is restricted via NetworkPolicy to DNS plus the specific downstream
  services required.
- TLS terminates at the ingress; in-cluster traffic uses ClusterIP services.

## Logging

All state-changing operations produce structured JSON logs to stdout, which
the cluster logging stack collects. Fields included by default:

- `time` (RFC3339)
- `level`
- `msg`
- `commit` (build SHA, injected at build time via ldflags)
- `component`
- Operation-specific fields: target resource, action, before/after state,
  dry-run indicator

A destructive operation logs the full before-state of the affected resource.

## Incident response

1. Capture the deployed image digest:
   `kubectl get pod -n <ns> <pod> -o jsonpath='{.spec.containers[0].image}'`
2. Pull recent logs: `kubectl logs -n <ns> deployment/myapp --tail=2000`
3. Filter for the failed operation: `... | jq 'select(.level == "ERROR")'`
4. Roll back: `git revert <commit>` and push, or scale the deployment to 0
   replicas to halt activity.
5. Open an issue linking the deployed image digest, the failing log entries,
   and the revert PR.

## Reporting a vulnerability

Report security issues privately through the repository's GitHub Security
Advisories ("Report a vulnerability" under the Security tab) rather than
opening a public issue.
