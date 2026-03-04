#!/bin/bash

# =============================================================================
# Release Committer
# =============================================================================
# Description: Stages and commits version changes for modules.
#              This script is typically run after pre-release.sh has
#              updated module versions. It creates a local commit only —
#              pushing, tagging, and Go proxy notification are handled by
#              prepare-release-pr.sh (Phase 1) and tag-release.sh (Phase 2).
#
# Usage: ./.github/scripts/release.sh [module]
#
# Arguments:
#   module           - Name of specific module to release (optional)
#                      If not provided, releases all modules with prepared versions
#
# Environment Variables:
#   DRY_RUN          - Enable dry run mode (default: true)
#                      When true, shows what would be committed and tagged without actually doing it
#
# Examples:
#   ./.github/scripts/release.sh
#   ./.github/scripts/release.sh container
#   DRY_RUN=false ./.github/scripts/release.sh
#   DRY_RUN=false ./.github/scripts/release.sh container
#
# Dependencies:
#   - git (configured with push permissions)
#   - go (for go.work parsing via 'go work edit -json')
#   - jq (for parsing go.work)
#
# Git Operations:
#   - Adds all modified version.go and go.mod files
#   - Creates commit with version bump message (e.g. chore(client): bump version)
#
# Note: This script no longer pushes to main, creates tags, or triggers the
#       Go proxy. Those operations are handled by the two-phase release process:
#       - Phase 1: prepare-release-pr.sh (creates a PR)
#       - Phase 2: tag-release.sh (tags after PR merge)
#
# =============================================================================

set -e

# Source common functions
readonly SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "${SCRIPT_DIR}/common.sh"

# Validate git remote before doing anything
validate_git_remote

MODULE="${1:-}"

# Collect and stage changes across modules, then create a single commit
if [[ -n "${MODULE}" ]]; then
  # Single module release
  echo "Releasing single module: ${MODULE}"
  commit_title="chore(${MODULE}): bump version"
else
  # All modules release
  echo "Releasing all modules with prepared versions"
  commit_title="chore(release): bump module versions"
fi

# Get all modules for staging go.mod changes
ALL_MODULES=$(get_modules)

commit_body=""

# Determine which modules to tag
if [[ -n "${MODULE}" ]]; then
  MODULES_TO_TAG="${MODULE}"
else
  MODULES_TO_TAG="${ALL_MODULES}"
fi

# Stage version.go and collect tag information only for modules being released.
# Note: Only version.go files for modules being released are staged here.
#       go.mod and go.sum files for all modules are staged separately below.
for m in $MODULES_TO_TAG; do
  next_tag_path=$(get_next_tag "${m}")
  # if the module version file does not exist, skip it
  if [[ ! -f "${next_tag_path}" ]]; then
    echo "Skipping ${m} because the pre-release script did not run"
    continue
  fi

  execute_or_echo git add "${ROOT_DIR}/${m}/version.go"

  nextTag=$(cat "${next_tag_path}")
  echo "Next tag for ${m}: ${nextTag}"
  commit_body="${commit_body}\n - ${m}: ${nextTag}"
done

# Stage go.mod and go.sum for ALL modules (they all need to reference the new version)
for m in $ALL_MODULES; do
  execute_or_echo git add "${ROOT_DIR}/${m}/go.mod"
  if [[ -f "${ROOT_DIR}/${m}/go.sum" ]]; then
    execute_or_echo git add "${ROOT_DIR}/${m}/go.sum"
  fi
done

if [[ "${DRY_RUN}" == "true" ]]; then
  echo ""
  echo "=========================================="
  echo "DRY RUN MODE - No git changes will be made"
  echo "=========================================="
  echo ""
  echo "Would create commit (local only, no push):"
  echo "  Title: ${commit_title}"
  echo "  Body: $(echo -e "${commit_body}")"
  echo ""
  echo "Files that would be committed:"
  for m in $MODULES_TO_TAG; do
    next_tag_path=$(get_next_tag "${m}")
    if [[ -f "${next_tag_path}" ]]; then
      echo "  ${m}/version.go"
    fi
  done
  for m in $ALL_MODULES; do
    echo "  ${m}/go.mod"
    if [[ -f "${ROOT_DIR}/${m}/go.sum" ]]; then
      echo "  ${m}/go.sum"
    fi
  done
  echo ""
  echo "Changes in module files:"
  for m in $ALL_MODULES; do
    echo ""
    echo "--- ${m}/... ---"
    git --no-pager diff "${ROOT_DIR}/${m}" || echo "  (new file)"
  done
  echo ""
  echo "=========================================="
  echo "NOTE: This script only creates a local commit."
  echo "Tags and pushing are handled by the two-phase release process."
  echo "See RELEASING.md for details."
  echo ""
  echo "To perform the actual commit, run:"
  echo "  DRY_RUN=false $0 $@"
  echo "=========================================="
  exit 0
fi

# Create a single commit if there are staged changes
if [[ -n "$(git diff --cached)" ]]; then
  execute_or_echo git commit -m "${commit_title}" -m "$(echo -e "${commit_body}")"
else
  echo "No changes detected in modules. Release process aborted."
  exit 1 # exit with error code 1 to not proceed with the release
fi

echo ""
echo "✅ Created commit successfully"
echo "Last commit:"
git_log_format='%C(auto)%h%C(reset) %s%nAuthor: %an <%ae>%nDate:   %ad'
execute_or_echo git -C "${ROOT_DIR}" --no-pager log -1 --pretty=format:"${git_log_format}" --date=iso-local
echo ""

echo ""
echo "=========================================="
echo "NOTE: This script no longer pushes directly to main or creates tags."
echo "Use the two-phase release process instead:"
echo "  Phase 1: ./.github/scripts/prepare-release-pr.sh — creates a release PR"
echo "  Phase 2: ./.github/scripts/tag-release.sh — auto-tags after PR merge"
echo "See RELEASING.md for details."
echo "=========================================="
