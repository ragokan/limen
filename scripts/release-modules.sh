#!/usr/bin/env sh
set -eu

version="${1:-}"
if [ "$version" = "" ]; then
	echo "usage: $0 vX.Y.Z" >&2
	exit 2
fi

case "$version" in
	v*) ;;
	*) version="v${version}" ;;
esac

if [ -n "$(git status --short)" ]; then
	echo "working tree must be clean" >&2
	exit 1
fi

./scripts/check-internal-module-versions.sh "$version"
./scripts/test-modules.sh --race
./scripts/test-standalone-modules.sh --all --local-replace

tags=""
for dir in $(./scripts/list-modules.sh); do
	if [ "$dir" = "." ]; then
		tag="$version"
	else
		tag="${dir#./}/${version}"
	fi

	if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
		echo "tag already exists locally: ${tag}" >&2
		exit 1
	fi
	if git ls-remote --exit-code --tags origin "refs/tags/${tag}" >/dev/null 2>&1; then
		echo "tag already exists on origin: ${tag}" >&2
		exit 1
	fi
	tags="${tags} ${tag}"
done

for tag in $tags; do
	echo "tag ${tag}"
	if [ "${DRY_RUN:-0}" != "1" ]; then
		git tag -a "$tag" -m "Release $tag"
	fi
done

if [ "${PUSH:-0}" = "1" ] && [ "${DRY_RUN:-0}" != "1" ]; then
	git push origin $tags
fi
