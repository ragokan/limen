#!/usr/bin/env sh
set -eu

module_args=""
race=""
local_replace="0"
script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repo_root="$(dirname "$script_dir")"

while [ "$#" -gt 0 ]; do
	case "$1" in
		--all)
			module_args="--all"
			;;
		--race)
			race="-race"
			;;
		--local-replace)
			local_replace="1"
			;;
		*)
			echo "usage: $0 [--all] [--race] [--local-replace]" >&2
			exit 2
			;;
	esac
	shift
done

for dir in $("$repo_root/scripts/list-modules.sh" ${module_args}); do
	echo "=== standalone ${dir} ==="
	if [ "$local_replace" = "1" ]; then
		tmpdir="$(mktemp -d)"
		trap 'rm -rf "$tmpdir"' EXIT INT TERM
		cp "$repo_root/$dir/go.mod" "$tmpdir/go.mod"
		if [ -f "$repo_root/$dir/go.sum" ]; then
			cp "$repo_root/$dir/go.sum" "$tmpdir/go.sum"
		fi
		for replace_dir in $("$repo_root/scripts/list-modules.sh"); do
			module_path="$(awk '/^module / {print $2; exit}' "$repo_root/$replace_dir/go.mod")"
			module_abs="$(cd "$repo_root/$replace_dir" && pwd)"
			(cd "$repo_root/$dir" && go mod edit -modfile="$tmpdir/go.mod" -replace="${module_path}=${module_abs}")
		done
		(cd "$repo_root/$dir" && GOWORK=off go test ${race} -modfile="$tmpdir/go.mod" -mod=mod -count=1 ./...)
		rm -rf "$tmpdir"
		trap - EXIT INT TERM
	else
		(cd "$repo_root/$dir" && GOWORK=off go test ${race} -mod=mod -count=1 ./...)
	fi
done
