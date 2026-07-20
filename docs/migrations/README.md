# Migration and rollout plans

Any change above moderate complexity -- schema migrations, data backfills,
cutovers, multi-phase rollouts -- gets a plan document in this directory
before implementation starts. The plan is the audit trail: when execution
deviates, the doc records what actually happened, not what was hoped.

## Conventions

- Filename: `YYYY-MM-DD_short-slug.md`, dated by creation.
- Header block, first lines of the doc:
  - `Created:` creation date
  - `State:` notstarted | started | complete. Update as the plan moves, and
    note what changed in parentheses (PRs merged, phases executed, images
    deployed).
  - `Issue:` the tracking issue, when one exists
  - `Depends on:` decision IDs (D-###) and earlier plans this builds on
  - `Reserves:` decision IDs this plan will add to
    [docs/DECISIONS.md](../DECISIONS.md), claimed up front so parallel work
    does not collide on numbers
- Body sections: Problem, Design (creating the reserved D-### entries),
  Validation, and Appendix A (below).
- Phases that touch production data get executed one at a time, each gated on
  the previous phase's appendix checks passing.

## Appendix A -- the validation tracker

Every phase that changes data records, in the doc itself:

1. A BEFORE query with its output, run against the real target before the
   change. This is the baseline.
2. An AFTER query with its output, run after the change. Predict the expected
   AFTER values in the plan before executing, so the AFTER run is a check
   against a stated expectation, not an observation rationalized after the
   fact.

Queries live in the appendix as fenced SQL with recorded outputs as trailing
comments. If a check comes back different from the prediction, the phase
stops and the discrepancy goes in the doc before any remediation.

## Skeleton

```markdown
# Title: what changes and why it is safe

Created: YYYY-MM-DD
State: notstarted
Issue: #NNN
Depends on: D-0XX, YYYY-MM-DD_prior-plan.md
Reserves: D-0YY

## Problem

What is wrong or missing, with measured baseline numbers.

## Design (D-0YY)

The decision this plan implements, alternatives rejected, phase list.

## Validation

How each phase proves itself: tests, differential checks, deployed-surface
spot checks.

## Appendix A -- validation tracker

BEFORE: recorded query + output.
AFTER (predicted): expected values.
AFTER (recorded): actual query + output, date, environment.
```
