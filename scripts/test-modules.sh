#!/usr/bin/env sh
set -eu

module_args=""
race=""

while [ "$#" -gt 0 ]; do
	case "$1" in
		--all)
			module_args="--all"
			;;
		--race)
			race="-race"
			;;
		*)
			echo "usage: $0 [--all] [--race]" >&2
			exit 2
			;;
	esac
	shift
done

for dir in $(./scripts/list-modules.sh ${module_args}); do
	echo "=== testing ${dir} ==="
	(cd "$dir" && go test ${race} -count=1 ./...)
done
