#!/usr/bin/env sh
set -eu

compose_file="benchmarks/docker-compose.yml"

docker compose -f "$compose_file" up -d --wait postgres
trap 'docker compose -f "$compose_file" down -v' EXIT INT TERM

export LIMEN_POSTGRES_DSN="${LIMEN_POSTGRES_DSN:-postgres://limen:limen@127.0.0.1:55432/limen_test?sslmode=disable}"
export BENCH_NAME="${BENCH_NAME:-ci-$(git rev-parse --short HEAD)}"
export BENCH_COUNT="${BENCH_COUNT:-3}"

./scripts/run-benchmarks.sh
