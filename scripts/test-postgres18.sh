#!/usr/bin/env sh
set -eu

compose_file="benchmarks/docker-compose.yml"
dsn="${LIMEN_POSTGRES_DSN:-postgres://limen:limen@127.0.0.1:55432/limen_test?sslmode=disable}"

docker compose -f "$compose_file" up -d --wait postgres
trap 'docker compose -f "$compose_file" down -v' EXIT INT TERM

LIMEN_POSTGRES_DSN="$dsn" go test -count=1 ./adapters/sql ./adapters/gorm
