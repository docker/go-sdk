# Releasing Guide

This document describes how to perform releases for the Docker Go SDK project.

## Overview

The Docker Go SDK is a multi-module Go project organized as a workspace. Each module is versioned and released independently, but releases are typically coordinated across all modules.

Each module's version is defined in the `version.go` file at the root of the module.

## Release Process

### 1. GitHub Actions (Recommended)

The primary way to perform releases is through GitHub Actions workflow.

#### Navigate to Actions
1. Go to the [Actions tab](../../actions) in the GitHub repository
2. Select the "Release All Modules" workflow
3. Click "Run workflow"

#### Configure Release Parameters
- **Dry Run**: 
  - ✅ `true` (default) - Shows what would happen without making changes
  - ❌ `false` - Performs actual release with git commits and tags
- **Bump Type**: Select version increment type
  - `prerelease` (default) - Increments prerelease version (e.g., `v0.1.0-alpha005` → `v0.1.0-alpha006`)
  - `patch` - Increments patch version (e.g., `v0.1.0` → `v0.1.1`)
  - `minor` - Increments minor version (e.g., `v0.1.0` → `v0.2.0`)
  - `major` - Increments major version (e.g., `v0.1.0` → `v1.0.0`)

#### Release Steps
1. **First Run (Dry Run)**: Always start with `dry_run: true` to verify changes
   ```
   Dry Run: ✅ true
   Bump Type: prerelease
   ```
   
2. **Review Output**: Check the workflow logs to ensure:
   - Correct version increments
   - All modules are being updated
   - No unexpected changes

3. **Actual Release**: If dry run looks good, run again with `dry_run: false`
   ```
   Dry Run: ❌ false  
   Bump Type: prerelease
   ```

### 2. Manual Release (Advanced)

If you need to perform releases manually or troubleshoot issues:

#### Prerequisites
- Docker installed (for semver-tool)
- jq installed (`brew install jq` on macOS)
- Git configured with push permissions
- All modules building successfully

#### Commands
```bash
# Dry run to see what would happen
DRY_RUN=true make release-all

# Actual release
DRY_RUN=false make release-all
```

#### Environment Variables
- `DRY_RUN`: `true` (default) or `false`. It generates the release (commit and tags) locally, without pushing changes to the remote repository.
- `BUMP_TYPE`: `prerelease` (default), `patch`, `minor`, or `major`. To know more about the bump type values, please read more [here](https://github.com/fsaintjacques/semver-tool).

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

### 1. Version Calculation
- Finds latest tag for each module
- Uses semver-tool to calculate next version
- Handles prerelease numbering with leading zeros

### 2. File Updates
For each module:
- Writes the next version to a file in the build directory, located at `.github/scripts/.build/<module>-next-tag`. This is a temporary file that is used to store the next version for the module.
- Updates `<module>/version.go` with new version
- Updates all `go.mod` files with new cross-module dependencies
- Runs `go mod tidy` to update `go.sum` files

### 3. Git Operations
- Commits all version changes
- Creates git tags for each module (e.g., `client/v0.1.0-alpha006`)
- Pushes changes and tags to GitHub

If `DRY_RUN` is `true`, the script does not push changes and tags to the remote repository.

### 4. Go Proxy Registration
- Triggers Go proxy to fetch new module versions
- Makes modules immediately available for download

If `DRY_RUN` is `true`, the script does not trigger the Go proxy.

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

If a release fails partway through:

1. **Reset changes**: `git reset --hard HEAD`
2. **Check current state**: `git status`
3. **Review logs**: Check GitHub Actions logs for specific errors
4. **Fix issues**: Address any underlying problems
5. **Retry**: Run release process again

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
