# micro-optimizations Results

Environment:

- Go: `go1.26.4 darwin/arm64`
- CPU: Apple M4 Pro
- Branch: `micro-optimizations`
- Samples: `-count=10` for root before/after benchmarks
- Baseline: `main` plus the benchmark harness only
- Latest Compose images tested: `postgres:latest` -> PostgreSQL `18.4`, `redis:latest` -> Redis `8.8.0`

Root benchmark comparison:

- Geomean: `-24.47%` across root benchmarks.
- Router static route: `301.9 ns/op` -> `273.8 ns/op` (`-9.29%`), allocations `12` -> `9`.
- Router param route: `436.9 ns/op` -> `452.6 ns/op` (`+3.60%`), allocations `15` -> `13`.
- Router with middleware: `484.5 ns/op` -> `429.8 ns/op` (`-11.29%`), allocations `18` -> `13`.
- Validator email: `2380.7 ns/op` -> `210.7 ns/op` (`-91.15%`), allocations `67` -> `1`.
- Database find-many: `15.63 us/op` -> `15.50 us/op` (`-0.80%`).
- Rate limiter same key: `951.1 ns/op` -> `884.8 ns/op` (`-6.97%`), allocations `16` -> `12`.
- Rate limiter new keys: `637.1 ns/op` -> `569.0 ns/op` (`-10.70%`), allocations `11` -> `7`.

The original rate-limiter new-key slowdown came from an unbounded per-key mutex
map: global map lock, map insert, mutex allocation, and retained keys for every
new rate-limit key. Fixed lock stripes preserve same-key correctness without
that allocation and growth path.

Session store comparison, Docker Compose `postgres:latest` + `redis:latest`,
`-count=5`:

| Benchmark | PostgreSQL | Redis | Result |
| --- | ---: | ---: | --- |
| Set | `1.588 ms/op`, `1955 B/op`, `28 allocs/op` | `200.1 us/op`, `2646 B/op`, `72 allocs/op` | Redis `7.94x` faster |
| Get | `75.68 us/op`, `2960 B/op`, `63 allocs/op` | `79.37 us/op`, `1216 B/op`, `21 allocs/op` | Redis `4.88%` slower |
| ListByUserID, 100 sessions | `224.7 us/op`, `168841 B/op`, `3133 allocs/op` | `6.356 ms/op`, `122227 B/op`, `1423 allocs/op` | PostgreSQL `28.28x` faster |
| DeleteByUserID, 10 sessions | `2.061 ms/op`, `267 B/op`, `6 allocs/op` | `710.1 us/op`, `13885 B/op`, `180 allocs/op` | Redis `2.90x` faster |

Redis session storage offloads session TTL to Redis key expiry; PostgreSQL stores
`expires_at` and depends on request-time validation or cleanup. The current Redis
`ListByUserID` path is intentionally visible in the results: it keeps a JSON user
index and checks each token key, which is slower than PostgreSQL indexed lookup
for user session listing.

Adapter benchmark snapshots:

- SQL adapter, SQLite in-memory: find-one `9.060 us/op`, find-many `60.50 us/op`, create `8.460 us/op`.
- GORM adapter, SQLite in-memory: find-one `7.478 us/op`, find-many `123.3 us/op`, create `7.804 us/op`.
- Redis cache adapter, Docker Compose Redis `8.8.0`: set+get `128.2 us/op`, `488 B/op`, `16 allocs/op`.

Verification:

- Non-example module loop passed: `find . -name go.mod -not -path './examples/*' ... go test -count=1 ./...`.
- Example module loop passed: `find examples -name go.mod ... go test -run '^$' ./...`.
- Root full lint passed: `golangci-lint run ./...`.
- Root full lint with integration benchmark tag passed: `golangci-lint run --build-tags bench_integration ./...`.
- Standalone OAuth module lint passed: `golangci-lint run ./...` in `plugins/oauth`.
- Standalone generic OAuth module lint passed: `golangci-lint run ./...` in `plugins/oauth-generic`.
- Branch-diff lint passed for every non-example module: `golangci-lint run --new-from-rev=main ./...`.
- PostgreSQL 18 integration passed for `adapters/sql` and `adapters/gorm` using `postgres:18`.
- Redis adapter integration and benchmark passed using `redis:8-alpine`.
- Docker Compose latest-image verification passed: `./scripts/test-compose-latest.sh`.
- Docker Compose session store comparison passed: `./scripts/bench-session-stores.sh`.
- Race checks passed for rate limiter and cache-session concurrent index tests.

Raw files:

- [micro-optimizations-baseline-root.txt](micro-optimizations-baseline-root.txt)
- [micro-optimizations-after-root-latest.txt](micro-optimizations-after-root-latest.txt)
- [micro-optimizations-after-sql.txt](micro-optimizations-after-sql.txt)
- [micro-optimizations-after-gorm.txt](micro-optimizations-after-gorm.txt)
- [micro-optimizations-benchstat-root.txt](micro-optimizations-benchstat-root.txt)
- [micro-optimizations-redis.txt](micro-optimizations-redis.txt)
- [micro-optimizations-rate-limiter-striped.txt](micro-optimizations-rate-limiter-striped.txt)
- [session-store-postgres-vs-redis.txt](session-store-postgres-vs-redis.txt)
- [latest-compose-tests-benchmarks.txt](latest-compose-tests-benchmarks.txt)
