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

Run PostgreSQL 18 integration tests:

```bash
./scripts/test-postgres18.sh
```

Run CI-style benchmarks with Docker Compose and PostgreSQL 18:

```bash
./scripts/ci-benchmarks.sh
```

When `LIMEN_POSTGRES_DSN` is set, SQL and GORM adapter benchmark suites include
PostgreSQL-backed cases in addition to their SQLite-backed cases.

Raw results for this branch are in [results](results/).
The checked-in `micro-optimizations` files are historical branch snapshots, not
release-wide performance guarantees.
The [next-steps summary](results/next-steps-summary.md) records the smoke run
used to verify the PostgreSQL benchmark harness.
