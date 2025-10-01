#!/bin/bash

# =============================================================================
# Release Finalizer
# =============================================================================
# Description: Commits and tags version changes for all modules, then triggers
#              Go proxy to make the new versions available for download
#              This script is typically run after pre-release.sh has
#              updated all module versions
#
# Usage: ./.github/scripts/release.sh
#
# Environment Variables:
#   DRY_RUN          - Enable dry run mode (default: true)
#                      When true, shows git commands without executing them
#
# Examples:
#   ./.github/scripts/release.sh
#   DRY_RUN=false ./.github/scripts/release.sh
#
# Dependencies:
#   - git (configured with push permissions)
#   - jq (for parsing go.work)
#   - curl (for triggering Go proxy)
#
# Git Operations:
#   - Adds all modified version.go and go.mod files
#   - Creates commit with version bump message (e.g. chore(client): bump version to v0.1.0-alpha005)
#   - Creates tag with module name and version (e.g. client/v0.1.0-alpha005)
#   - Pushes changes and tags to origin
#
# Post-Release Operations:
#   - Triggers Go proxy to fetch new module versions
#   - Makes modules immediately available for download
#
# =============================================================================

set -e

# Source common functions
readonly SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "${SCRIPT_DIR}/common.sh"

# Collect and stage changes across modules, then create a single commit
MODULES=$(go work edit -json | jq -r '.Use[] | "\(.DiskPath | ltrimstr("./"))"' | tr '\n' ' ' && echo)

commit_title="chore(release): bump module versions"
commit_body=""

for m in $MODULES; do
  # if the module version file does not exist, skip it
  if [[ ! -f "${ROOT_DIR}/.github/scripts/.${m}-next-tag" ]]; then
    echo "Skipping ${m} because the pre-release script did not run"
    continue
  fi

  execute_or_echo git add "${ROOT_DIR}/${m}/version.go"
  execute_or_echo git add "${ROOT_DIR}/${m}/go.mod"
  if [[ -f "${ROOT_DIR}/${m}/go.sum" ]]; then
    execute_or_echo git add "${ROOT_DIR}/${m}/go.sum"
  fi

  nextTag=$(cat "${ROOT_DIR}/.github/scripts/.${m}-next-tag")
  echo "Next tag for ${m}: ${nextTag}"
  commit_body="${commit_body}\n - ${m}: ${nextTag}"
done

# Create a single commit if there are staged changes
if [[ -n "$(git diff --cached)" ]]; then
  execute_or_echo git commit -m "${commit_title}" -m "$(echo -e "${commit_body}")"
else
  echo "No staged changes to commit"
fi

# Create all tags after the single commit
for m in $MODULES; do
  if [[ -f "${ROOT_DIR}/.github/scripts/.${m}-next-tag" ]]; then
    nextTag=$(cat "${ROOT_DIR}/.github/scripts/.${m}-next-tag")
    execute_or_echo git tag "${m}/${nextTag}"
  fi
done

execute_or_echo git push origin main --tags

for m in $MODULES; do
  nextTag=$(cat "${ROOT_DIR}/.github/scripts/.${m}-next-tag")
  curlGolangProxy "${m}" "${nextTag}"
  execute_or_echo rm "${ROOT_DIR}/.github/scripts/.${m}-next-tag"
done
