#!/usr/bin/env sh
set -eu

name="${LIMEN_PG_CONTAINER:-limen-pg18}"
port="${LIMEN_PG_PORT:-55432}"
dsn="postgres://limen:limen@127.0.0.1:${port}/limen_test?sslmode=disable"

docker rm -f "$name" >/dev/null 2>&1 || true
docker run --name "$name" \
	-e POSTGRES_USER=limen \
	-e POSTGRES_PASSWORD=limen \
	-e POSTGRES_DB=limen_test \
	-p "${port}:5432" \
	-d postgres:18 >/dev/null

cleanup() {
	docker rm -f "$name" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

until docker exec "$name" pg_isready -U limen -d limen_test >/dev/null 2>&1; do
	sleep 1
done

LIMEN_POSTGRES_DSN="$dsn" go test -count=1 ./adapters/sql ./adapters/gorm
