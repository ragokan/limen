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
2. Run validation without creating tags:

```bash
DRY_RUN=1 ./scripts/release-modules.sh vX.Y.Z
```

3. Create and push annotated tags:

```bash
PUSH=1 ./scripts/release-modules.sh vX.Y.Z
```

The script requires a clean worktree, verifies internal module requirements,
runs workspace module tests, runs standalone module tests with `GOWORK=off` and
local module replaces, and creates the root/submodule tag set. CI separately
checks public standalone resolution after tags are published.

## GitHub Release

Use the `Release` workflow with `dry_run=true` first. Run it again with
`dry_run=false` to push tags from CI.

After tags exist, create a GitHub release for the root tag and include the
notable changes, security notes, and verification commands.

## Versioning

Before `v1.0.0`, public APIs may still change. Keep adapter interfaces, provider
interfaces, cleanup APIs, schema configuration, and hook behavior stable within
a minor line unless a security fix requires a breaking change.
