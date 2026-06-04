# Benchmarks

This directory stores repeatable benchmark commands and raw result files for
performance work.

Run the root and selected module benchmarks:

```bash
./scripts/run-benchmarks.sh
```

Run with more samples:

```bash
BENCH_COUNT=10 ./scripts/run-benchmarks.sh
```

Name a run:

```bash
BENCH_NAME=release-v0.1.6 BENCH_COUNT=10 ./scripts/run-benchmarks.sh
```

Run PostgreSQL 18 integration tests:

```bash
./scripts/test-postgres18.sh
```

Run CI-style benchmarks with Docker Compose and PostgreSQL 18:

```bash
./scripts/ci-benchmarks.sh
```

Dispatch the GitHub benchmark workflow:

```bash
gh workflow run benchmarks.yml --repo ragokan/limen --ref main
```

When `LIMEN_POSTGRES_DSN` is set, SQL and GORM adapter benchmark suites include
PostgreSQL-backed cases in addition to their SQLite-backed cases.

Raw results for this branch are in [results](results/).
The checked-in `micro-optimizations` files are historical branch snapshots, not
release-wide performance guarantees.
The [next-steps summary](results/next-steps-summary.md) records the smoke run
used to verify the PostgreSQL benchmark harness.
The [CI benchmark snapshot](results/ci-26961254493-summary.md) records the
GitHub Actions workflow verification run.

See [Benchmarking](../docs/benchmarking.md) for workflow and result guidance.

## Comparing Runs

Install `benchstat` when you need statistical comparison:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

Then compare raw result files:

```bash
benchstat benchmarks/results/baseline-root.txt benchmarks/results/candidate-root.txt
```

Use a clean worktree for release-facing benchmark manifests. Dirty-worktree
results are acceptable for smoke checks, but should not be used for release
claims.
