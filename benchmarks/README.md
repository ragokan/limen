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

Run the Docker Compose PostgreSQL vs Redis session-store comparison:

```bash
./scripts/bench-session-stores.sh
```

The Compose file pulls `postgres:latest` and `redis:latest` by default. Override
with `LIMEN_POSTGRES_IMAGE` or `LIMEN_REDIS_IMAGE` when a pinned image is needed.

Run adapter integration tests and Redis adapter benchmarks against the same
latest-image Compose stack:

```bash
./scripts/test-compose-latest.sh
```

Redis adapter tests and benchmarks are opt-in:

```bash
docker run --rm -d --name limen-redis-bench -p 56379:6379 redis:8-alpine
LIMEN_REDIS_ADDR=127.0.0.1:56379 go test ./adapters/redis
LIMEN_REDIS_ADDR=127.0.0.1:56379 go test -run '^$' -bench . -benchmem ./adapters/redis
docker stop limen-redis-bench
```

Raw results for this branch are in [results](results/).
