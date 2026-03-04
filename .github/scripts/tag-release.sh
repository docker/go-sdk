#!/bin/bash

# =============================================================================
# Tag Release (Phase 2)
# =============================================================================
# Description: Creates git tags from version.go files and triggers Go proxy.
#              Runs after a release PR is merged to main.
#              This is Phase 2 of the two-phase release process.
#
# Usage: ./.github/scripts/tag-release.sh [module]
#
# Arguments:
#   module           - Name of specific module to tag (optional)
#                      If not provided, tags all modules with unreleased versions
#
# Key Properties:
#   - Derives versions from version.go (no dependency on .build/ files)
#   - Idempotent: existing tags are skipped
#   - Squash-merge safe: tags the current HEAD (merge commit)
#
# Dependencies:
#   - git (configured with push permissions)
#   - go (for go.work parsing via 'go work edit -json')
#   - jq (for parsing go.work)
#   - curl (for triggering Go proxy)
#
# =============================================================================

set -eo pipefail

# Source common functions
readonly SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "${SCRIPT_DIR}/common.sh"

# Validate git remote before doing anything
validate_git_remote

# Ensure we're on main — tags must point to commits reachable from main
CURRENT_BRANCH=$(git -C "${ROOT_DIR}" rev-parse --abbrev-ref HEAD)
if [[ "${CURRENT_BRANCH}" != "main" ]]; then
  echo "❌ Error: Must be on the 'main' branch to create release tags"
  echo "  Current branch: ${CURRENT_BRANCH}"
  echo ""
  echo "Switch to main first:"
  echo "  git checkout main"
  exit 1
fi

MODULE="${1:-}"

echo "=== Phase 2: Tag Release ==="
echo ""

# Determine which modules to process
if [[ -n "${MODULE}" ]]; then
  MODULES_TO_TAG="${MODULE}"
else
  MODULES_TO_TAG=$(get_modules)
fi

tags_created=0
tags_skipped=0

for m in $MODULES_TO_TAG; do
  VERSION_FILE="${ROOT_DIR}/${m}/version.go"

  if [[ ! -f "${VERSION_FILE}" ]]; then
    if [[ -n "${MODULE}" ]]; then
      echo "❌ Error: Module '${m}' was specified, but ${VERSION_FILE} does not exist"
      exit 1
    fi
    echo "⚠️  Skipping ${m}: version.go not found"
    continue
  fi

  # Read version from version.go
  VERSION=$(get_version_from_file "${VERSION_FILE}")
  if [[ -z "${VERSION}" ]]; then
    if [[ -n "${MODULE}" ]]; then
      echo "❌ Error: Module '${m}' was specified, but could not extract version from ${VERSION_FILE}"
      exit 1
    fi
    echo "⚠️  Skipping ${m}: could not extract version from version.go"
    continue
  fi

  # Ensure version has v prefix
  if [[ ! "${VERSION}" =~ ^v ]]; then
    VERSION="v${VERSION}"
  fi

  TAG="${m}/${VERSION}"

  # Check if tag already exists on remote (the authoritative source)
  # Use fixed-string match (-F) to avoid regex issues with dots in version numbers
  if git ls-remote --tags origin 2>/dev/null | grep -Fq "refs/tags/${TAG}"; then
    echo "⏭️  Skipping ${m}: tag ${TAG} already exists on remote"
    tags_skipped=$((tags_skipped + 1))
    continue
  fi

  # If the tag exists locally but not on remote (e.g., previous run failed to push),
  # verify it points to HEAD and push it; otherwise recreate it
  if git tag --list | grep -Fxq "${TAG}"; then
    LOCAL_TAG_SHA=$(git rev-parse "${TAG}" 2>/dev/null)
    HEAD_SHA=$(git rev-parse HEAD)
    if [[ "${LOCAL_TAG_SHA}" == "${HEAD_SHA}" ]]; then
      echo "🔁 Tag ${TAG} exists locally but not on remote — pushing existing tag"
      execute_or_echo git push origin "${TAG}"
    else
      echo "⚠️  Local tag ${TAG} points to ${LOCAL_TAG_SHA}, not HEAD (${HEAD_SHA}) — recreating"
      execute_or_echo git tag -f "${TAG}"
      execute_or_echo git push origin "${TAG}"
    fi
  else
    # Create tag on HEAD and push it
    echo "🏷️  Creating tag: ${TAG}"
    execute_or_echo git tag "${TAG}"
    execute_or_echo git push origin "${TAG}"
  fi
  tags_created=$((tags_created + 1))

  # Trigger Go proxy
  echo "📦 Triggering Go proxy for ${m}@${VERSION}..."
  curlGolangProxy "${m}" "${VERSION}"
  echo ""
done

echo ""
echo "=== Tag Release Summary ==="
echo "  Tags created: ${tags_created}"
echo "  Tags skipped (already exist): ${tags_skipped}"

if [[ ${tags_created} -gt 0 ]]; then
  echo ""
  echo "✅ Tags created and pushed successfully!"
  echo "Tags on HEAD:"
  git --no-pager tag --list --points-at HEAD
elif [[ ${tags_skipped} -gt 0 ]]; then
  echo ""
  echo "✅ All tags already exist — nothing to do (idempotent)."
else
  echo ""
  echo "⚠️  No modules were processed."
fi
