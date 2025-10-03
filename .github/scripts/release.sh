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
#                      When true, shows commands without executing them, except for git commands
#                      that are executed but before pushing the changes to the remote repository.
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
  next_tag_path=$(get_next_tag "${m}")
  # if the module version file does not exist, skip it
  if [[ ! -f "${next_tag_path}" ]]; then
    echo "Skipping ${m} because the pre-release script did not run"
    continue
  fi

  git add "${ROOT_DIR}/${m}/version.go"
  git add "${ROOT_DIR}/${m}/go.mod"
  if [[ -f "${ROOT_DIR}/${m}/go.sum" ]]; then
    git add "${ROOT_DIR}/${m}/go.sum"
  fi

  nextTag=$(cat "${next_tag_path}")
  echo "Next tag for ${m}: ${nextTag}"
  commit_body="${commit_body}\n - ${m}: ${nextTag}"
done

# Create a single commit if there are staged changes
if [[ -n "$(git diff --cached)" ]]; then
  git commit -m "${commit_title}" -m "$(echo -e "${commit_body}")"
else
  echo "No changes detected in modules. Release process aborted."
  exit 1 # exit with error code 1 to not proceed with the release
fi

# Create all tags after the single commit
for m in $MODULES; do
  next_tag_path=$(get_next_tag "${m}")
  if [[ -f "${next_tag_path}" ]]; then
    nextTag=$(cat "${next_tag_path}")
    git tag "${m}/${nextTag}"
  fi
done

if [[ "${DRY_RUN}" == "true" ]]; then
  echo "Remote operations will be skipped."
  # show the last commit, including the patch
  echo "Last commit:"
  git_log_format='%C(auto)%h%C(reset) %s%nAuthor: %an <%ae>%nDate:   %ad'
  git -C "${ROOT_DIR}" --no-pager log -1 --pretty=format:"${git_log_format}" --date=iso-local
  git -C "${ROOT_DIR}" --no-pager show -1 --format= --patch --stat
  # list the new tags, that should point to the same last commit
  echo "New tags:"
  git -C "${ROOT_DIR}" --no-pager tag --list --points-at HEAD
fi

echo "Pushing changes and tags to remote repository..."

execute_or_echo git push origin main --tags

for m in $MODULES; do
  nextTag=$(cat $(get_next_tag "${m}"))
  curlGolangProxy "${m}" "${nextTag}"
done
