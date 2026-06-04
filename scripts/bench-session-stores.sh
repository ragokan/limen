#!/usr/bin/env sh
set -eu

compose_file="benchmarks/docker-compose.yml"
count="${BENCH_COUNT:-5}"
out_dir="${BENCH_OUT_DIR:-benchmarks/results}"
out="$out_dir/session-store-postgres-vs-redis.txt"

mkdir -p "$out_dir"

docker compose -f "$compose_file" up -d --wait
trap 'docker compose -f "$compose_file" down -v >/dev/null 2>&1' EXIT INT TERM

{
	echo "# Session store PostgreSQL vs Redis"
	date
	go version
	git rev-parse --short HEAD
	git status --short
	echo
	docker compose -f "$compose_file" ps
	echo
	LIMEN_POSTGRES_DSN="${LIMEN_POSTGRES_DSN:-postgres://limen:limen@127.0.0.1:55432/limen_bench?sslmode=disable}" \
		LIMEN_REDIS_ADDR="${LIMEN_REDIS_ADDR:-127.0.0.1:56379}" \
		go test -tags bench_integration -run '^TestSessionStoreRedisTTLExpires$' -count=1 .
	echo
	LIMEN_POSTGRES_DSN="${LIMEN_POSTGRES_DSN:-postgres://limen:limen@127.0.0.1:55432/limen_bench?sslmode=disable}" \
		LIMEN_REDIS_ADDR="${LIMEN_REDIS_ADDR:-127.0.0.1:56379}" \
		go test -tags bench_integration -run '^$' -bench '^BenchmarkSessionStore' -benchmem -count="$count" .
} | tee "$out"
