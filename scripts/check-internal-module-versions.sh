#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repo_root="$(dirname "$script_dir")"

version="${1:-}"
if [ "$version" = "" ]; then
	version="$(cat "$repo_root/VERSION")"
fi

case "$version" in
	v*) ;;
	*) version="v${version}" ;;
esac

tmp="${TMPDIR:-/tmp}/limen-internal-version-check.$$"
trap 'rm -f "$tmp"' EXIT INT TERM

find "$repo_root" -name go.mod -not -path "$repo_root/.git/*" -print | sort | while read -r file; do
	awk -v version="$version" '
		/^[[:space:]]*github\.com\/ragokan\/limen(\/[^[:space:]]*)?[[:space:]]+/ {
			if ($2 != version) {
				printf "%s:%d: expected %s, got %s\n", FILENAME, FNR, version, $0
			}
		}
		/^require[[:space:]]+github\.com\/ragokan\/limen(\/[^[:space:]]*)?[[:space:]]+/ {
			if ($3 != version) {
				printf "%s:%d: expected %s, got %s\n", FILENAME, FNR, version, $0
			}
		}
	' "$file"
done > "$tmp"

if [ -s "$tmp" ]; then
	cat "$tmp"
	exit 1
fi
