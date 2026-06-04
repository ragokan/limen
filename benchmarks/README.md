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

Raw results for this branch are in [results](results/).
