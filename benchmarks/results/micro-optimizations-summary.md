# micro-optimizations Results

Environment:

- Go: `go1.26.4 darwin/arm64`
- CPU: Apple M4 Pro
- Branch: `micro-optimizations`
- Samples: `-count=10` for root before/after benchmarks
- Baseline: `main` plus the benchmark harness only

Root benchmark comparison:

- Geomean: `-24.47%` across root benchmarks.
- Router static route: `301.9 ns/op` -> `273.8 ns/op` (`-9.29%`), allocations `12` -> `9`.
- Router param route: `436.9 ns/op` -> `452.6 ns/op` (`+3.60%`), allocations `15` -> `13`.
- Router with middleware: `484.5 ns/op` -> `429.8 ns/op` (`-11.29%`), allocations `18` -> `13`.
- Validator email: `2380.7 ns/op` -> `210.7 ns/op` (`-91.15%`), allocations `67` -> `1`.
- Database find-many: `15.63 us/op` -> `15.50 us/op` (`-0.80%`).
- Rate limiter same key: `951.1 ns/op` -> `884.8 ns/op` (`-6.97%`), allocations `16` -> `12`.
- Rate limiter new keys: `637.1 ns/op` -> `569.0 ns/op` (`-10.70%`), allocations `11` -> `7`.

PostgreSQL cleanup:

- `Limen.CleanupExpired(ctx)` deletes expired sessions, verifications, and database-backed static-window rate-limit rows.
- Cleanup runs once after initialization by default.
- Disable initialization cleanup with `Cleanup: limen.NewDefaultCleanupConfig(limen.WithCleanupOnInit(false))`.
- `ListSessions(ctx, userID)` now returns only non-expired database sessions.

Adapter benchmark snapshots:

- SQL adapter, SQLite in-memory: find-one `9.060 us/op`, find-many `60.50 us/op`, create `8.460 us/op`.
- GORM adapter, SQLite in-memory: find-one `7.478 us/op`, find-many `123.3 us/op`, create `7.804 us/op`.

Verification:

- Root tests passed: `go test -count=1 ./...`.
- Root full lint passed: `golangci-lint run ./...`.
- Root full lint with integration benchmark tag passed: `golangci-lint run --build-tags bench_integration ./...`.
- Standalone OAuth module lint passed: `golangci-lint run ./...` in `plugins/oauth`.
- Standalone generic OAuth module lint passed: `golangci-lint run ./...` in `plugins/oauth-generic`.
- PostgreSQL 18 integration passed for `adapters/sql` and `adapters/gorm`.
- Race checks passed for rate limiter and cache-session concurrent index tests.

Raw files:

- [micro-optimizations-baseline-root.txt](micro-optimizations-baseline-root.txt)
- [micro-optimizations-after-root-latest.txt](micro-optimizations-after-root-latest.txt)
- [micro-optimizations-benchstat-root.txt](micro-optimizations-benchstat-root.txt)
- [micro-optimizations-rate-limiter-striped.txt](micro-optimizations-rate-limiter-striped.txt)
