# Decision Log

Format: D-###, date, status (accepted | proposed | superseded). Keep entries
short; link the discussion or plan doc when one exists. Record a decision the
moment it is made -- an unrecorded decision gets relitigated. When a plan doc
in docs/migrations/ will introduce decisions, it declares the numbers it
reserves so parallel work does not collide.

An entry says what was decided, the one or two live alternatives that were
rejected, and why. It does not restate the code.

## D-001 -- Scaffolded from repo_template_go (REPLACE-DATE, accepted)

This service was instantiated from repo_template_go: cobra + slog CLI, stdlib
net/http server with probes and graceful shutdown, distroless digest-pinned
container, golangci-lint v2, pre-commit with a pre-push build/test tier, and
the docs conventions in this directory (this log, docs/migrations/ plan docs,
docs/development/ doctrine). Deviations from the template's opinions get
their own D-### entries rather than silent divergence.
