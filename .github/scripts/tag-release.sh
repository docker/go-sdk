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
#   - jq (for parsing go.work)
#   - curl (for triggering Go proxy)
#
# =============================================================================

set -e

# Source common functions
readonly SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "${SCRIPT_DIR}/common.sh"

# Validate git remote before doing anything
validate_git_remote

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
    echo "⚠️  Skipping ${m}: version.go not found"
    continue
  fi

  # Read version from version.go
  VERSION=$(get_version_from_file "${VERSION_FILE}")
  if [[ -z "${VERSION}" ]]; then
    echo "⚠️  Skipping ${m}: could not extract version from version.go"
    continue
  fi

  # Ensure version has v prefix
  if [[ ! "${VERSION}" =~ ^v ]]; then
    VERSION="v${VERSION}"
  fi

  TAG="${m}/${VERSION}"

  # Check if tag already exists (locally or remotely)
  if git tag --list | grep -q "^${TAG}$" || git ls-remote --tags origin "${TAG}" 2>/dev/null | grep -q "${TAG}"; then
    echo "⏭️  Skipping ${m}: tag ${TAG} already exists"
    tags_skipped=$((tags_skipped + 1))
    continue
  fi

  # Create tag on HEAD and push it individually
  echo "🏷️  Creating tag: ${TAG}"
  git tag "${TAG}"
  git push origin "${TAG}"
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
