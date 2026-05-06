# Security

This document describes the security controls, data flow, and compliance mapping for `myapp`.

> **Template note**: replace `myapp`, the data-classification table, the integrations, and the architecture diagram with values that match the actual service. Sections marked `[REPLACE]` are placeholders.

## Compliance Context

`myapp` operates within the following compliance frameworks:

| Framework | Relevance |
|-----------|-----------|
| HIPAA     | [REPLACE: describe how this service touches PHI, or state explicitly that it does not] |
| SOC 2     | Access control, audit logging, change management |
| ISO 27001 | Information security management, access control policy enforcement |
| HITRUST   | [REPLACE: relevance for health information security] |

If this service does not handle PHI, replace this section with a positive statement to that effect rather than leaving it ambiguous.

## Data Classification

| Data | Classification | Storage |
|---|---|---|
| [REPLACE: input config / payloads] | [REPLACE: PII / PHI / public] | [REPLACE] |
| Audit logs | Operational metadata; treat as PHI-adjacent if the service touches PHI | stdout to cluster log aggregation |
| Secrets (API keys, webhook secrets) | Restricted | External secret manager, mounted as env vars |

## Architecture and Data Flow

```
[REPLACE WITH ACTUAL DIAGRAM]

  Client / upstream system
        |
        v
  +-----------+
  |   myapp   |  <-- env-injected secrets
  +-----------+
        |
        v
  Downstream API / database
        |
        v
  Structured JSON audit logs (stdout -> cluster logging)
```

## Access Controls

### Service Authentication

| Target | Method | Secret Source |
|--------|--------|--------------|
| [REPLACE: downstream API] | [REPLACE: bearer / mTLS / OIDC] | [REPLACE: external secret manager] |

### Principle of Least Privilege

- The service account has the minimum K8s RBAC required for operation.
- Secrets are scoped to the service's namespace and not accessible cross-namespace.
- No direct database access unless explicitly required; prefer API-mediated mutations.

## Audit Trail

All state-changing operations produce structured JSON logs to stdout. Logs are collected by the cluster logging stack and retained per organisational policy.

Log fields included by default:
- `time` (RFC3339)
- `level`
- `msg`
- `commit` (build SHA, injected at build time via ldflags)
- `component`
- Operation-specific fields: target resource, action, before/after state, dry-run indicator

## Secret Management

```
External Secret Manager (source of truth)
        |
        | ExternalSecrets / CSI driver
        v
K8s Secret (in service's namespace)
        |
        | env var injection
        v
myapp container
```

- Secrets never appear in git, CI logs, or container images.
- Access to the secret manager is via workload identity federation (no static credentials).

## Network Security

- The service is reachable only via internal cluster networking unless an explicit ingress is configured.
- Egress is restricted via NetworkPolicy to DNS plus the specific downstream services required.
- TLS termination is at the ingress; in-cluster traffic uses ClusterIP services.

## Container Hardening

- Multi-stage build; runtime image is `gcr.io/distroless/static-debian12:nonroot` with no shell or package manager.
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

## Destructive Operations

[REPLACE: enumerate the destructive operations this service can perform and the safeguards in place. If the service is read-only, state that.]

- Dry-run mode: [REPLACE: how to preview changes without mutating]
- Change review: [REPLACE: PR / approval requirements]
- Audit logging: every destructive operation logs the full before-state of the affected resource.

## Incident Response

1. Capture the deployed image digest: `kubectl get pod -n <ns> <pod> -o jsonpath='{.spec.containers[0].image}'`
2. Pull recent logs: `kubectl logs -n <ns> deployment/myapp --tail=2000`
3. Filter for the failed operation: `... | jq 'select(.level == "ERROR")'`
4. Roll back: `git revert <commit>` and push, or scale the deployment to 0 replicas to halt activity.
5. Open an incident ticket and link the deployed image digest, the failing log entries, and the revert PR.
