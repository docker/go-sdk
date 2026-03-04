#!/bin/bash

# =============================================================================
# Prepare Release PR (Phase 1)
# =============================================================================
# Description: Creates a release branch, runs pre-release for target module(s),
#              stages changes, commits, pushes the branch, and creates a PR.
#              This is Phase 1 of the two-phase release process.
#
# Usage: ./.github/scripts/prepare-release-pr.sh [module]
#
# Arguments:
#   module           - Name of specific module to release (optional)
#                      If not provided, releases all modules
#
# Environment Variables:
#   BUMP_TYPE        - Type of version bump (default: prerelease)
#
# Dependencies:
#   - git (configured with push permissions, origin must point to docker/go-sdk)
#   - gh (GitHub CLI, for creating PRs)
#   - jq (for parsing go.work)
#   - Docker (for semver-tool, used by pre-release.sh)
#
# =============================================================================

set -e

# Source common functions
readonly SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "${SCRIPT_DIR}/common.sh"

# Validate git remote before doing anything
validate_git_remote

MODULE="${1:-}"
BUMP_TYPE="${BUMP_TYPE:-prerelease}"
TIMESTAMP="$(date +%Y%m%d%H%M%S)"

# Determine branch name and commit title
if [[ -n "${MODULE}" ]]; then
  BRANCH_NAME="release/bump-${MODULE}-${TIMESTAMP}"
  COMMIT_TITLE="chore(${MODULE}): bump version"
else
  BRANCH_NAME="release/bump-versions-${TIMESTAMP}"
  COMMIT_TITLE="chore(release): bump module versions"
fi

# Ensure we start from a clean, up-to-date main branch
CURRENT_BRANCH=$(git -C "${ROOT_DIR}" rev-parse --abbrev-ref HEAD)
if [[ "${CURRENT_BRANCH}" != "main" ]]; then
  echo "❌ Error: Must be on the 'main' branch to create a release PR"
  echo "  Current branch: ${CURRENT_BRANCH}"
  echo ""
  echo "Switch to main first:"
  echo "  git checkout main"
  exit 1
fi

if [[ -n "$(git -C "${ROOT_DIR}" status --porcelain)" ]]; then
  echo "❌ Error: Working tree is not clean"
  echo "  Commit or stash your changes before running a release."
  exit 1
fi

echo "Fetching latest from origin..."
git -C "${ROOT_DIR}" fetch origin main
LOCAL_SHA=$(git -C "${ROOT_DIR}" rev-parse HEAD)
REMOTE_SHA=$(git -C "${ROOT_DIR}" rev-parse origin/main)
if [[ "${LOCAL_SHA}" != "${REMOTE_SHA}" ]]; then
  echo "❌ Error: Local main is not up to date with origin/main"
  echo "  Local:  ${LOCAL_SHA}"
  echo "  Remote: ${REMOTE_SHA}"
  echo ""
  echo "Update your local main first:"
  echo "  git pull origin main"
  exit 1
fi

echo "=== Phase 1: Prepare Release PR ==="
echo "  Module: ${MODULE:-all}"
echo "  Bump type: ${BUMP_TYPE}"
echo "  Branch: ${BRANCH_NAME}"
echo ""

# Create release branch from up-to-date main
git checkout -b "${BRANCH_NAME}"

# Clean build directory
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Run pre-release for target module(s)
if [[ -n "${MODULE}" ]]; then
  echo "Running pre-release for module: ${MODULE}"
  env DRY_RUN=false BUMP_TYPE="${BUMP_TYPE}" "${SCRIPT_DIR}/pre-release.sh" "${MODULE}"
else
  echo "Running pre-release for all modules..."
  MODULES=$(get_modules)
  for m in $MODULES; do
    echo ""
    echo "--- Pre-releasing module: ${m} ---"
    env DRY_RUN=false BUMP_TYPE="${BUMP_TYPE}" "${SCRIPT_DIR}/pre-release.sh" "${m}"
  done
fi

# Get all modules for staging
ALL_MODULES=$(get_modules)

# Determine which modules to include in version summary
if [[ -n "${MODULE}" ]]; then
  MODULES_TO_TAG="${MODULE}"
else
  MODULES_TO_TAG="${ALL_MODULES}"
fi

# Stage version.go files for released modules and build commit body
commit_body=""
for m in $MODULES_TO_TAG; do
  next_tag_path=$(get_next_tag "${m}")
  if [[ ! -f "${next_tag_path}" ]]; then
    echo "Skipping ${m} because the pre-release script did not run"
    continue
  fi

  git add "${ROOT_DIR}/${m}/version.go"
  nextTag=$(cat "${next_tag_path}")
  commit_body="${commit_body}\n - ${m}: ${nextTag}"
done

# Stage go.mod and go.sum for ALL modules
for m in $ALL_MODULES; do
  git add "${ROOT_DIR}/${m}/go.mod"
  if [[ -f "${ROOT_DIR}/${m}/go.sum" ]]; then
    git add "${ROOT_DIR}/${m}/go.sum"
  fi
done

# Verify there are staged changes
if [[ -z "$(git diff --cached)" ]]; then
  echo "No changes detected. Aborting."
  exit 1
fi

# Commit
git commit -m "${COMMIT_TITLE}" -m "$(echo -e "${commit_body}")"

# Push the branch
git push origin "${BRANCH_NAME}"

# Build PR body
PR_BODY="## Release Version Bump

**Bump type**: \`${BUMP_TYPE}\`

### Version changes:
$(echo -e "${commit_body}")

---
This PR was created automatically by the release workflow.
Merging this PR will trigger Phase 2 (automatic tagging and Go proxy update)."

# Create PR with gh
PR_URL=$(gh pr create \
  --title "${COMMIT_TITLE}" \
  --body "${PR_BODY}" \
  --base main \
  --head "${BRANCH_NAME}" \
  --label "chore" \
  2>&1) || {
    echo "Warning: gh pr create failed. The branch has been pushed."
    echo "You can create the PR manually from: ${BRANCH_NAME}"
    echo "Error: ${PR_URL}"
    exit 1
  }

echo ""
echo "✅ Release PR created successfully!"
echo "  PR: ${PR_URL}"
echo ""
echo "Next steps:"
echo "  1. Review the PR"
echo "  2. Merge it to trigger Phase 2 (automatic tagging)"
