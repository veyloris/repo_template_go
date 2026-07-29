# Benchmark baseline

`baseline.txt` is the committed reference run for `benchstat`. It is checked
in on purpose: when a change moves a benchmark, the new numbers land in the
pull request diff, where a reviewer has to look at them and the author has to
justify them. A baseline kept only on a developer's machine gets quietly
re-recorded and the regression is never seen.

**There is no `baseline.txt` yet.** The template ships no benchmarks, so
there is nothing to record. Add the first benchmark with its written
acceptance target (see
[docs/development/testing.md](../../docs/development/testing.md)), then run
`task bench-baseline` and commit the file it writes.

## Recording and comparing

```bash
task bench            # every benchmark, -count 6, into bench-current.txt (gitignored)
task bench-compare    # benchstat baseline.txt vs a fresh run
task bench-baseline   # overwrite baseline.txt -- commit the diff, explain any move
```

Re-record only when the move is understood and intended: a real optimization,
a deliberate trade, or a change in the fixture scale. Re-recording to make a
red number go away discards the only regression signal there is.

## What this file is and is not

These are one machine's absolute numbers. They are not a service-level
objective, not a production latency, and not divisible into a request's
duration -- a benchmark taken on a developer CPU can be several times faster
than the same code on a throttled container, so the ratio is meaningless.
Use the baseline for relative comparison of versions of the same code, and
use production histograms to attribute where a request's time goes.

## Name it `.txt`, never `.out`

`.gitignore` ignores `*.out` for coverage profiles. A baseline named
`baseline.out` would be silently ignored, never committed, and every
`benchstat` invocation in CI or on a fresh clone would fail on a missing
file -- or worse, be "fixed" by re-recording it locally each time, which
removes the comparison entirely. The extension is load-bearing.
