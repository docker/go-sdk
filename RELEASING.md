# Releasing Guide

This document describes how to perform releases for the Docker Go SDK project.

## Overview

The Docker Go SDK is a multi-module Go project organized as a workspace. Each module is versioned and released independently, but releases are typically coordinated across all modules.

Each module's version is defined in the `version.go` file at the root of the module.

## Two-Phase Release Process

Releases follow a **two-phase process** that uses pull requests instead of direct pushes to `main`:

- **Phase 1 — Prepare Release PR**: A workflow bumps versions and creates a PR with all changes.
- **Phase 2 — Auto-Tag on Merge**: When the PR is merged to `main`, tags are automatically created on the merge commit and the Go proxy is notified.

This ensures that `main` always reflects released versions, tags point to reachable commits, and all changes go through code review.

## Phase 1: Prepare a Release PR

### Via GitHub Actions (Recommended)

1. Go to the [Actions tab](../../actions) in the GitHub repository
2. Select the **"Release"** workflow
3. Click **"Run workflow"**
4. Configure release parameters:
   - **Module**: Leave empty to release all modules, or enter a module name (e.g., `client`, `container`)
   - **Bump Type**: `prerelease` (default), `patch`, `minor`, or `major`
   - **Dry Run**: `true` (default) — preview changes without creating a PR
5. **First Run (Dry Run)**: Always start with `dry_run: true` to preview version changes
6. **Review Output**: Check the workflow logs for version increments
7. **Create PR**: If satisfied, run again with `dry_run: false` to create the release PR

The workflow will:
- Create a `release/bump-*` branch
- Run `pre-release.sh` for the target module(s)
- Commit all `version.go`, `go.mod`, and `go.sum` changes
- Push the branch and create a PR with the `chore` label

### Releasing a Single Module

Use the same **"Release"** workflow but enter the module name in the **Module** field:

```
Module: container
Bump Type: patch
Dry Run: false
```

The module name is validated against the modules in `go.work`.

### Local Dry Run Preview

Always start with a dry run. This does not require origin to point to `docker/go-sdk` — it only previews version changes without any git operations:

```bash
# Preview version changes for all modules
DRY_RUN=true make pre-release-all

# Preview for a specific module
cd container
DRY_RUN=true make pre-release
```

### Running Phase 1 Locally

After reviewing the dry run output, you can run the release PR script directly from your machine. Your `origin` remote **must** point to `docker/go-sdk`:

```bash
BUMP_TYPE=prerelease ./.github/scripts/prepare-release-pr.sh          # all modules
BUMP_TYPE=prerelease ./.github/scripts/prepare-release-pr.sh client   # single module
```

The script will:
1. Validate that `origin` points to `docker/go-sdk` (fails with instructions if not)
2. Verify you're on `main` with a clean working tree
3. Fetch `origin/main` and verify your local branch is up to date
4. Create a release branch, bump versions, commit, push, and open a PR

## Phase 2: Automatic Tagging

Phase 2 runs **automatically** when a push to `main` modifies any `*/version.go` file (i.e., when a release PR is merged).

### Safety Guards

Phase 2 has two layers of protection to prevent accidental tagging:

1. **Path filter**: The workflow only triggers on pushes that modify `*/version.go` files.
2. **Commit message check**: The tagging step only proceeds if the commit message matches the release pattern produced by `prepare-release-pr.sh` (`chore(release): bump module versions` or `chore(<module>): bump version`). This prevents non-release PRs that happen to touch `version.go` from creating tags.

The commit message check works with all merge strategies:
- **Squash merge**: PR title becomes the commit subject — matches directly
- **Regular merge**: PR title appears in the merge commit body — matched by grep
- **Rebase merge**: Original commit message is preserved — matches directly

### What tag-release.sh Does

For each module:
1. Reads the version from `version.go`
2. Checks if tag `<module>/v<version>` already exists (locally and on remote)
3. Creates the tag on HEAD (the merge commit) if it doesn't exist
4. Pushes each tag individually
5. Triggers the Go proxy to index the new version

### Key Properties

- **No dependency on `.build/` files** — derives everything from `version.go` vs existing git tags
- **Idempotent** — existing tags are skipped; safe to re-run
- **Squash-merge safe** — tags the merge commit, not the original branch commit

## Manual Tagging (Advanced)

If Phase 2 fails or you need to re-tag manually, you can run `tag-release.sh` directly. Your `origin` remote must point to `docker/go-sdk`:

```bash
# Tag all modules (from the repository root, on main)
DRY_RUN=false make tag-release

# Tag a specific module
cd client
DRY_RUN=false make tag-release
```

The script is idempotent — it skips tags that already exist.

## Environment Variables

- `DRY_RUN`: `true` (default) or `false`
  - `true`: Shows what would be done without making any changes
  - `false`: Creates commits, PRs, tags, etc.
- `BUMP_TYPE`: `prerelease` (default), `patch`, `minor`, or `major`
  - Controls how the version number is incremented
  - Read more about semver [here](https://github.com/fsaintjacques/semver-tool)
## Release Types

### Prerelease
- **Purpose**: Development versions, testing, early access
- **Version Format**: `v0.1.0-alpha001`, `v0.1.0-alpha002`, etc.
- **Naming**: Uses 3-digit zero-padded increments
- **Stability**: No API stability guarantees

### Patch Release
- **Purpose**: Bug fixes, security updates
- **Version Format**: `v0.1.0` → `v0.1.1`
- **Compatibility**: Backwards compatible

### Minor Release
- **Purpose**: New features, backwards compatible changes
- **Version Format**: `v0.1.0` → `v0.2.0`
- **Compatibility**: Backwards compatible

### Major Release
- **Purpose**: Breaking changes, major API changes
- **Version Format**: `v0.1.0` → `v1.0.0`
- **Compatibility**: May include breaking changes

## Troubleshooting

### Orphaned Tags (Tags Without Corresponding Main Commit)

If tags were pushed but `main` doesn't contain the version bump commit:

1. Create a PR from the branch/commit that has the version changes, targeting `main`
2. Merge the PR
3. Phase 2 fires, sees the tags already exist, and skips them (idempotent)
4. State is now consistent: `main` has the version changes, tags exist

**Do NOT delete existing tags** — the Go proxy has already indexed them and the community may depend on them.

### Origin Remote Points to a Fork

Both `prepare-release-pr.sh` and `tag-release.sh` validate that `origin` points to `docker/go-sdk`. If you see:

```
❌ Error: Git remote 'origin' points to the wrong repository
```

Fix it:
```bash
git remote set-url origin git@github.com:docker/go-sdk.git
```

### Re-running Phase 2

If Phase 2 fails or you need to re-tag:

```bash
# From the repository root on main
DRY_RUN=false make tag-release

# Or for a specific module
cd client
DRY_RUN=false make tag-release
```

### Common Issues

#### "No such file or directory" errors
- Ensure you're running from the repository root
- Check that all modules exist and have `version.go` files

#### "Permission denied" on git operations
- Verify git is configured with push permissions
- Check GitHub token has appropriate permissions

#### Version calculation errors
- Verify Docker is installed and accessible
- Check that semver-tool image can be pulled: `docker pull mdelapenya/semver-tool:3.4.0`

#### Go mod tidy failures
- Ensure Go is installed and configured
- Check that all modules compile independently

#### PR creation fails
- Ensure `gh` CLI is installed and authenticated (`gh auth status`)
- Check that the `chore` label exists in the repository

### Getting Help

- Check GitHub Actions logs for detailed error messages
- Review this document for common issues
- Examine shell scripts in `.github/scripts/` for implementation details

## Best Practices

1. **Always dry run first** — Use `dry_run: true` to verify changes
2. **Test before releasing** — Ensure all tests pass
3. **Review the release PR** — Check version increments and dependency updates
4. **Monitor after release** — Check that modules are available on [pkg.go.dev](https://pkg.go.dev/github.com/docker/go-sdk)

## Security Considerations

- GitHub Actions are pinned to specific commit SHAs
- Secrets are handled through GitHub's secure environment
- All operations are logged and auditable
- Dry run mode prevents accidental releases
- All version changes go through PR review before tagging
- Origin remote is validated to prevent pushing to wrong repository
