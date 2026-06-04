#!/usr/bin/env sh
set -eu

compose_file="benchmarks/docker-compose.yml"
out_dir="${BENCH_OUT_DIR:-benchmarks/results}"
out="$out_dir/latest-compose-tests-benchmarks.txt"

mkdir -p "$out_dir"

docker compose -f "$compose_file" up -d --wait
trap 'docker compose -f "$compose_file" down -v >/dev/null 2>&1' EXIT INT TERM

{
	echo "# Docker Compose latest tests and benchmarks"
	date
	go version
	echo "Postgres:"
	docker run --rm "${LIMEN_POSTGRES_IMAGE:-postgres:latest}" postgres --version
	echo "Redis:"
	docker run --rm "${LIMEN_REDIS_IMAGE:-redis:latest}" redis-server --version
	echo
	docker compose -f "$compose_file" ps
	echo
	echo "## PostgreSQL adapter tests"
	LIMEN_POSTGRES_DSN="${LIMEN_POSTGRES_DSN:-postgres://limen:limen@127.0.0.1:55432/limen_bench?sslmode=disable}" \
		go test -count=1 ./adapters/sql ./adapters/gorm
	echo
	echo "## Redis adapter tests"
	LIMEN_REDIS_ADDR="${LIMEN_REDIS_ADDR:-127.0.0.1:56379}" \
		go test -count=1 ./adapters/redis
	echo
	echo "## Redis adapter benchmarks"
	LIMEN_REDIS_ADDR="${LIMEN_REDIS_ADDR:-127.0.0.1:56379}" \
		go test -run '^$' -bench . -benchmem -count="${BENCH_COUNT:-5}" ./adapters/redis
} | tee "$out"
