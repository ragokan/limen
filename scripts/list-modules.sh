#!/usr/bin/env sh
set -eu

mode="publishable"
if [ "${1:-}" = "--all" ]; then
	mode="all"
fi

if [ "$mode" = "all" ]; then
	find . -name go.mod -not -path './.git/*' -exec dirname {} \; | sort
else
	find . -name go.mod -not -path './.git/*' -not -path './examples/*' -exec dirname {} \; | sort
fi
