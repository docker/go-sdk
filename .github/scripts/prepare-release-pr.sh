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
#   DRY_RUN          - Enable dry run mode (default: true)
#                      When true, expansion + version preview run but no branch
#                      is created and no commit/push/PR happens. Set to "false"
#                      to actually create the release PR.
#
# Dependencies:
#   - git (configured with push permissions, origin must point to docker/go-sdk)
#   - go (for go.work parsing and go mod tidy)
#   - gh (GitHub CLI, for creating PRs; only required when DRY_RUN=false)
#   - jq (for parsing go.work)
#   - Docker (for semver-tool, used by pre-release.sh)
#
# =============================================================================

set -eo pipefail

readonly SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "${SCRIPT_DIR}/common.sh"

MODULE=$(echo "${1:-}" | tr '[:upper:]' '[:lower:]')
BUMP_TYPE="${BUMP_TYPE:-prerelease}"
TIMESTAMP="$(date +%Y%m%d%H%M%S)"

# Always validate the module name (cheap, surfaces typos in both run modes).
# Two-step validation:
#   1. Format check: must be a single token of [a-z0-9-]+ — guards against
#      whitespace (grep -F treats embedded newlines in the pattern as alternation,
#      which would silently make a newline-separated MODULE match any of its parts)
#      and against regex/glob metacharacters.
#   2. Existence check: grep -Fxq for exact-line, fixed-string matching against
#      the modules listed in go.work.
if [[ -n "${MODULE}" ]]; then
  if ! [[ "${MODULE}" =~ ^[a-z][a-z0-9-]*$ ]]; then
    echo "❌ Error: Module name must match [a-z][a-z0-9-]* — got '${MODULE}'"
    exit 1
  fi

  ALL_MODS=$(get_modules)
  if ! printf '%s\n' ${ALL_MODS} | grep -Fxq -- "${MODULE}"; then
    echo "❌ Error: Module '${MODULE}' not found in go.work"
    echo ""
    echo "Available modules:"
    for m in ${ALL_MODS}; do echo "  - ${m}"; done
    exit 1
  fi
fi

# Real-run pre-flight: origin must be docker/go-sdk, working tree must be a
# clean main that's in sync with origin. Dry runs skip these so the preview
# works from any branch, fork, or detached state — handy for contributors
# who haven't set their origin to docker/go-sdk.
if [[ "${DRY_RUN}" != "true" ]]; then
  validate_git_remote

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
fi

# Compute the modules to release. When releasing a single module, this also
# includes any in-repo module that requires it (transitively) — pre-release.sh
# already rewrites their go.mod, but we must also bump their version.go and
# tag them so main never drifts from the latest published tag.
MODULES_TO_RELEASE=$(get_modules_to_release "${MODULE}")
NUM_MODULES_TO_RELEASE=$(echo "${MODULES_TO_RELEASE}" | wc -w | tr -d ' ')

# Determine branch name and commit title.
# A single-module input that fans out to multiple modules switches to the
# release-wide title so Phase 2's commit-message check still matches.
if [[ -n "${MODULE}" && "${NUM_MODULES_TO_RELEASE}" -eq 1 ]]; then
  BRANCH_NAME="release/bump-${MODULE}-${TIMESTAMP}"
  COMMIT_TITLE="chore(${MODULE}): bump version"
else
  BRANCH_NAME="release/bump-versions-${TIMESTAMP}"
  COMMIT_TITLE="chore(release): bump module versions"
fi

echo "=== Phase 1: Prepare Release PR ==="
echo "  Module: ${MODULE:-all}"
echo "  Bump type: ${BUMP_TYPE}"
echo "  Branch: ${BRANCH_NAME}"
echo "  Modules to release: ${MODULES_TO_RELEASE}"
echo "  Dry run: ${DRY_RUN}"
echo ""

# Real run creates the release branch up front; dry run stays on main and
# never writes to the working tree (pre-release.sh is invoked with DRY_RUN=true).
if [[ "${DRY_RUN}" != "true" ]]; then
  git checkout -b "${BRANCH_NAME}"
fi

# Clean build directory so .build/<module>-next-tag reflects this run only
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Run pre-release for each module in the release set, propagating DRY_RUN
for m in ${MODULES_TO_RELEASE}; do
  echo ""
  echo "--- Pre-releasing module: ${m} ---"
  env DRY_RUN="${DRY_RUN}" BUMP_TYPE="${BUMP_TYPE}" "${SCRIPT_DIR}/pre-release.sh" "${m}"
done

# Build the version-summary commit body from the next-tag files that
# pre-release.sh wrote (these are produced in both dry and real runs).
commit_body=""
for m in ${MODULES_TO_RELEASE}; do
  next_tag_path=$(get_next_tag "${m}")
  if [[ ! -f "${next_tag_path}" ]]; then
    echo "Skipping ${m} because the pre-release script did not run"
    continue
  fi
  nextTag=$(cat "${next_tag_path}")
  commit_body="${commit_body}\n - ${m}: ${nextTag}"
done

if [[ "${DRY_RUN}" == "true" ]]; then
  echo ""
  echo "=== Dry Run Summary ==="
  echo -e "${commit_body}"
  echo ""
  echo "✅ Dry run completed. No git commits, pushes, or pull requests were created."
  echo "To create the release PR, re-run with DRY_RUN=false."
  exit 0
fi

# Real run: stage files, commit, push, open PR.

# Stage version.go for each released module
for m in ${MODULES_TO_RELEASE}; do
  next_tag_path=$(get_next_tag "${m}")
  if [[ -f "${next_tag_path}" ]]; then
    git add "${ROOT_DIR}/${m}/version.go"
  fi
done

# Stage go.mod and go.sum for ALL modules (pre-release.sh may have rewritten them)
ALL_MODULES=$(get_modules)
for m in $ALL_MODULES; do
  git add "${ROOT_DIR}/${m}/go.mod"
  if [[ -f "${ROOT_DIR}/${m}/go.sum" ]]; then
    git add "${ROOT_DIR}/${m}/go.sum"
  fi
done

if [[ -z "$(git diff --cached)" ]]; then
  echo "No changes detected. Aborting."
  exit 1
fi

git commit -m "${COMMIT_TITLE}" -m "$(echo -e "${commit_body}")"
git push origin "${BRANCH_NAME}"

PR_BODY="## Release Version Bump

**Bump type**: \`${BUMP_TYPE}\`

### Version changes:
$(echo -e "${commit_body}")

---
This PR was created automatically by the release workflow.
Merging this PR will trigger Phase 2 (automatic tagging and Go proxy update)."

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
