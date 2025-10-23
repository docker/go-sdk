# Releasing Guide

This document describes how to perform releases for the Docker Go SDK project.

## Overview

The Docker Go SDK is a multi-module Go project organized as a workspace. Each module is versioned and released independently, but releases are typically coordinated across all modules.

Each module's version is defined in the `version.go` file at the root of the module.

## Release Process

### 1. GitHub Actions (Recommended)

The primary way to perform releases is through GitHub Actions workflows.

**Important**: GitHub Actions release workflows can only be run from the `main` branch. The workflows will automatically be skipped if triggered from any other branch. This is a safety measure to ensure releases are only performed from the primary branch.

#### Releasing All Modules

1. Go to the [Actions tab](../../actions) in the GitHub repository
2. Select the "Release All Modules" workflow
3. Click "Run workflow"
4. Configure release parameters:
   - **Dry Run**: `true` (default) - Shows what would happen without making any git changes
   - **Bump Type**: `prerelease` (default), `patch`, `minor`, or `major`
5. **First Run (Dry Run)**: Always start with `dry_run: true` to preview changes
6. **Review Output**: Check the workflow logs for version increments and changes
7. **Actual Release**: If satisfied, run again with `dry_run: false`

#### Releasing a Single Module

1. Go to the [Actions tab](../../actions) in the GitHub repository
2. Select the "Release Single Module" workflow
3. Click "Run workflow"
4. Configure release parameters:
   - **Module**: Type the module name (e.g., `container`, `client`, `image`)
   - **Dry Run**: `true` (default) - Shows preview without making changes
   - **Bump Type**: `prerelease` (default), `patch`, `minor`, or `major`
5. Follow the same dry-run-first workflow as above

**Note**: The module name will be validated against the available modules in `go.work`. If you enter an invalid name, you'll get a helpful error message listing available modules.

### 2. Manual Release (Advanced)

If you need to perform releases manually or troubleshoot issues:

**Note**: Manual releases using `make` commands can technically be run from any branch, but should be run from the `main` branch for consistency with the GitHub Actions workflows.

#### Prerequisites
- Docker installed (for semver-tool)
- jq installed (`brew install jq` on macOS)
- Git configured with push permissions
- All modules building successfully

#### Releasing All Modules
```bash
# Step 1: Dry run to preview version changes (no build files created)
DRY_RUN=true make pre-release-all  # Explicit dry run, safe to run

# Step 2: Prepare release for real (creates build files in .github/scripts/.build/)
DRY_RUN=false make pre-release-all

# Step 3: Actual release (automatically checks pre-release, creates commits, tags, and pushes)
DRY_RUN=false make release-all

# With specific bump type
BUMP_TYPE=patch DRY_RUN=false make release-all
```

**Note**: The `release-all` target automatically runs `check-pre-release` for all modules to verify that `pre-release-all` was completed successfully (with `DRY_RUN=false`). If you try to run `release-all` without first running `pre-release-all` with `DRY_RUN=false`, it will fail with an error.

#### Releasing a Single Module
```bash
# From the module directory
cd container
make pre-release              # Prepare version files
DRY_RUN=true make release     # Preview (no git changes)
DRY_RUN=false make release    # Actual release

# Or using scripts directly from root
./.github/scripts/pre-release.sh container
DRY_RUN=false ./.github/scripts/release.sh container
```

#### Environment Variables
- `DRY_RUN`: `true` (default) or `false`
  - `true`: Shows what would be done without making any git changes (commits, tags, push)
  - `false`: Creates commits, tags, and pushes to remote
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

## What Happens During Release

### 1. Pre-Release Phase
The `pre-release-all` or `pre-release` command must be run first:
- Finds latest tag for each module
- Uses semver-tool to calculate next version
- Handles prerelease numbering with leading zeros
- Writes the next version to a file in the build directory, located at `.github/scripts/.build/<module>-next-tag`

### 2. Pre-Release Check
The `release-all` command automatically runs `check-pre-release` for all modules to verify:
- The `.github/scripts/.build` directory exists
- Each module has a corresponding `<module>-next-tag` file
- The version in the `<module>-next-tag` file matches the version in `<module>/version.go`
- If any checks fail, the release is aborted with an error message

This check is implemented in `.github/scripts/check-pre-release.sh` and ensures that `pre-release-all` was completed successfully (with `DRY_RUN=false`) and that all version files are properly updated before proceeding with the release.

You can manually run the check for a specific module:
```bash
make -C client check-pre-release
# or
cd client && make check-pre-release
```

### 3. File Updates
For each module:
- Updates `<module>/version.go` with new version
- Updates all `go.mod` files with new cross-module dependencies
- Runs `go mod tidy` to update `go.sum` files

### 4. Git Operations
When `DRY_RUN=false`:
- Creates a single commit with all version changes
- Creates git tags for each module (e.g., `client/v0.1.0-alpha006`)
- Pushes commit and tags to GitHub

When `DRY_RUN=true`:
- **No git operations are performed**
- Shows preview of what commit and tags would be created
- Shows diffs of version files
- Completely safe to run multiple times

### 5. Go Proxy Registration
When `DRY_RUN=false`:
- Triggers Go proxy to fetch new module versions
- Makes modules immediately available for download via `go get`

When `DRY_RUN=true`:
- No proxy registration occurs
- Preview only

## Troubleshooting

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

### Manual Recovery

#### If Pre-Release Succeeds but Release Fails

Since dry-run makes no git changes, you're always safe. If you want to start over:

```bash
# Remove the prepared version files and restore original state
cd container
git restore version.go go.mod go.sum
rm ../.github/scripts/.build/container-next-tag

# Then run pre-release again
make pre-release
```

#### If Release Partially Completes (DRY_RUN=false)

If a non-dry-run release fails partway through:

1. **Check current state**: `git status`
2. **Review what happened**: `git log -1` and `git tag --points-at HEAD`
3. **If commit was created but not pushed**:
   ```bash
   git reset --hard HEAD~1  # Undo commit
   git tag -d module/v0.1.0-alpha001  # Delete local tag
   ```
4. **If pushed to remote**: Contact maintainers - may need to create a follow-up release
5. **Retry**: Run release process again after fixes

### Getting Help

- Check GitHub Actions logs for detailed error messages
- Review this document for common issues
- Examine shell scripts in `.github/scripts/` for implementation details

## Best Practices

1. **Always dry run first** - Use `dry_run: true` to verify changes
2. **Test before releasing** - Ensure all tests pass
3. **Review version increments** - Verify the bump type is correct
4. **Monitor after release** - Check that modules are available on [pkg.go.dev](https://pkg.go.dev/github.com/docker/go-sdk)

## Security Considerations

- GitHub Actions are pinned to specific commit SHAs
- Secrets are handled through GitHub's secure environment
- All operations are logged and auditable
- Dry run mode prevents accidental releases
