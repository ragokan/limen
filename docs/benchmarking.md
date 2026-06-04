# Benchmarking

Limen keeps repeatable benchmark commands and raw snapshots under
[benchmarks](../benchmarks/).

## Local Benchmarks

Run all benchmark suites:

```bash
./scripts/run-benchmarks.sh
```

Run with PostgreSQL 18 through Docker Compose:

```bash
BENCH_COUNT=3 ./scripts/ci-benchmarks.sh
```

`ci-benchmarks.sh` starts PostgreSQL 18, sets `LIMEN_POSTGRES_DSN`, runs root,
SQL, GORM, OAuth, session JWT, and credential-password benchmarks, and writes
raw files to `benchmarks/results`.

## GitHub Benchmarks

The `Benchmarks` workflow is manual and scheduled weekly. It uses the same
Docker Compose PostgreSQL 18 harness and uploads the raw result files as the
`benchmark-results` artifact.

Manual dispatch:

```bash
gh workflow run benchmarks.yml --repo ragokan/limen --ref main
```

## Current Checked-In Snapshots

The checked-in snapshots are historical records, not release-wide guarantees.

- [micro-optimizations summary](../benchmarks/results/micro-optimizations-summary.md)
  records the optimization branch comparison.
- [next-steps summary](../benchmarks/results/next-steps-summary.md) records the
  PostgreSQL 18 smoke run for the benchmark harness.
- [CI benchmark snapshot 26961254493](../benchmarks/results/ci-26961254493-summary.md)
  records the GitHub Actions benchmark workflow verification run.

The PostgreSQL smoke snapshot used `BENCH_COUNT=1` and exists to prove the
harness, not to make final performance claims. Use `BENCH_COUNT=10` or higher
for decisions that depend on small deltas.

## Comparing Results

For release-facing comparisons:

1. Start from a clean worktree.
2. Run baseline and candidate benchmarks with the same `BENCH_COUNT`.
3. Compare matching raw files with `benchstat`.
4. Record the Go version, CPU, database image, sample count, and git commit.

```bash
go install golang.org/x/perf/cmd/benchstat@latest
benchstat benchmarks/results/baseline-root.txt benchmarks/results/candidate-root.txt
```

Dirty-worktree benchmark manifests are useful for smoke tests only.

## SQL Snapshot

From the PostgreSQL 18 smoke run:

| Operation | SQL PostgreSQL | GORM PostgreSQL |
| --- | ---: | ---: |
| `FindOne` | `70.948 us/op` | `77.232 us/op` |
| `FindMany` | `107.716 us/op` | `145.104 us/op` |
| `Create` | `1.174 ms/op` | `1.440 ms/op` |

In that run, the SQL adapter was about `8.1%` faster for `FindOne`, `25.8%`
faster for `FindMany`, and `18.5%` faster for `Create`.

## Root Snapshot

The ten-sample root comparison in the micro-optimizations branch recorded a
`-24.47%` geomean improvement across root benchmarks, with the largest win in
email validation (`2380.7 ns/op` to `210.7 ns/op`).
