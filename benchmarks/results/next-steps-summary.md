# Next Steps Benchmark Snapshot

Environment:

- Go: `go1.26.4 darwin/arm64`
- CPU: Apple M4 Pro
- PostgreSQL: `18.4` via Docker Compose (`postgres:18`)
- Samples: `BENCH_COUNT=1`
- Command: `BENCH_COUNT=1 BENCH_NAME=next-steps ./scripts/ci-benchmarks.sh`

This is a smoke benchmark for the new benchmark harness and PostgreSQL cases.
Use higher sample counts before making release-performance claims.

## Root

| Benchmark | Result |
| --- | ---: |
| `BenchmarkDatabaseFindOneUser` | `5.321 us/op` |
| `BenchmarkDatabaseFindManyUsers` | `14.135 us/op` |
| `BenchmarkDatabaseCreateUser` | `626.1 ns/op` |
| `BenchmarkRateLimiterCheckCacheSameKey` | `830.1 ns/op` |
| `BenchmarkRateLimiterCheckCacheNewKeys` | `462.5 ns/op` |
| `BenchmarkRouterStaticRoute` | `248.3 ns/op` |
| `BenchmarkRouterParamRoute` | `388.8 ns/op` |
| `BenchmarkRouterWithMiddleware` | `391.1 ns/op` |
| `BenchmarkValidatorEmail` | `205.3 ns/op` |
| `BenchmarkValidateJSONBodyLookup` | `9.516 ns/op` |

## Adapter Comparison

| Operation | SQL SQLite | GORM SQLite | SQL PostgreSQL | GORM PostgreSQL |
| --- | ---: | ---: | ---: | ---: |
| `FindOne` | `9.414 us/op` | `7.422 us/op` | `70.948 us/op` | `77.232 us/op` |
| `FindMany` | `60.337 us/op` | `123.545 us/op` | `107.716 us/op` | `145.104 us/op` |
| `Create` | `8.320 us/op` | `7.519 us/op` | `1.174 ms/op` | `1.440 ms/op` |

PostgreSQL snapshot:

- SQL adapter `FindOne` was about `8.1%` faster than GORM.
- SQL adapter `FindMany` was about `25.8%` faster than GORM.
- SQL adapter `Create` was about `18.5%` faster than GORM.

Raw files:

- [next-steps-manifest.txt](next-steps-manifest.txt)
- [next-steps-root.txt](next-steps-root.txt)
- [next-steps-sql.txt](next-steps-sql.txt)
- [next-steps-gorm.txt](next-steps-gorm.txt)
- [next-steps-oauth.txt](next-steps-oauth.txt)
- [next-steps-session-jwt.txt](next-steps-session-jwt.txt)
- [next-steps-credential-password.txt](next-steps-credential-password.txt)
