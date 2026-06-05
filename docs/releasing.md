# Releasing

Limen is a multi-module repository. A release tags the root module and each
publishable submodule at the same commit.

## Module Tags

Root:

```text
vX.Y.Z
```

Submodules:

```text
adapters/sql/vX.Y.Z
plugins/oauth/vX.Y.Z
cmd/limen/vX.Y.Z
```

Examples are not tagged.

## Local Release Flow

1. Update [VERSION](../VERSION) and internal `github.com/ragokan/limen...`
   requirements to the release version.
2. For performance-sensitive releases, refresh benchmarks from a clean
   worktree:

```bash
BENCH_NAME=release-vX.Y.Z BENCH_COUNT=10 ./scripts/ci-benchmarks.sh
```

3. Run validation without creating tags:

```bash
DRY_RUN=1 ./scripts/release-modules.sh vX.Y.Z
```

4. Create and push annotated tags:

```bash
PUSH=1 ./scripts/release-modules.sh vX.Y.Z
```

The script requires a clean worktree, verifies internal module requirements,
runs workspace module tests, runs standalone module tests with `GOWORK=off` and
local module replaces, and creates the root/submodule tag set. After tags are
published, run standalone module tests without local replaces to verify public
module resolution.

## GitHub Release

Use the `Release` workflow with `dry_run=true` first. The validation job runs
with read-only repository permissions and prints the planned module tags without
creating or pushing them.

Run it again with `dry_run=false` from the default branch to push tags from CI.
The publish job has write permissions, depends on the validation job, and only
runs when the workflow is dispatched from the repository default branch.

After tags exist, create a GitHub release for the root tag and include the
notable changes, security notes, and verification commands.

## Versioning

Before `v1.0.0`, public APIs may still change. Keep adapter interfaces, provider
interfaces, cleanup APIs, schema configuration, and hook behavior stable within
a minor line unless a security fix requires a breaking change.
