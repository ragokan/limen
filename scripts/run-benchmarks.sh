#!/usr/bin/env sh
set -eu

count="${BENCH_COUNT:-5}"
out_dir="${BENCH_OUT_DIR:-benchmarks/results}"
name="${BENCH_NAME:-micro-optimizations}"
mkdir -p "$out_dir"

manifest="$out_dir/${name}-manifest.txt"
{
	echo "# Benchmark manifest"
	date
	go version
	git rev-parse --short HEAD
	git status --short
	uname -a
	if [ "${LIMEN_POSTGRES_DSN:-}" != "" ]; then
		echo "LIMEN_POSTGRES_DSN=${LIMEN_POSTGRES_DSN}"
		docker compose -f benchmarks/docker-compose.yml exec -T postgres postgres --version
	fi
} | tee "$manifest"

run_one() {
	dir="$1"
	label="$2"
	out="$out_dir/${name}-${label}.txt"
	{
		echo "# Benchmark ${label}"
		date
		go version
		git rev-parse --short HEAD
		git status --short
		echo
		(cd "$dir" && go test -run '^$' -bench . -benchmem -count="$count" ./...)
	} | tee "$out"
}

run_one "." "root"
run_one "./adapters/sql" "sql"
run_one "./adapters/gorm" "gorm"
run_one "./plugins/oauth" "oauth"
run_one "./plugins/session-jwt" "session-jwt"
run_one "./plugins/credential-password" "credential-password"
