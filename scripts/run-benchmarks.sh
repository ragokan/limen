#!/usr/bin/env sh
set -eu

count="${BENCH_COUNT:-5}"
out_dir="${BENCH_OUT_DIR:-benchmarks/results}"
name="${BENCH_NAME:-micro-optimizations}"
mkdir -p "$out_dir"

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
