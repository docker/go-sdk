#!/bin/bash

# =============================================================================
# Pre-Release Version Updater
# =============================================================================
# Description: Updates version numbers in Go modules for releases
#              This script bumps the version for a specific module and
#              updates all related go.mod files across the repository
#
# Usage: ./.github/scripts/pre-release.sh <module>
#
# Arguments:
#   module           - Name of the module to update (required)
#                      Examples: client, container, config, context, image, network
#
# Environment Variables:
#   BUMP_TYPE        - Type of version bump (default: prerelease)
#                      Options: prerelease, patch, minor, major
#   DRY_RUN          - Enable dry run mode (default: true)
#                      When true, shows what would be done without making changes
#
# Examples:
#   ./.github/scripts/pre-release.sh client
#   BUMP_TYPE=patch ./.github/scripts/pre-release.sh container
#   DRY_RUN=false ./.github/scripts/pre-release.sh config
#
# Dependencies:
#   - Docker (for semver-tool)
#   - jq (for parsing go.work)
#   - git (for tag operations)
#
# Files Modified:
#   - <module>/version.go    - Updates version constant
#   - */go.mod               - Updates module dependencies
#   - */go.sum               - Updated via go mod tidy
#
# Version Logic:
#   - Handles both prerelease (with leading zeros) and final versions
#   - Automatically detects if version should have prerelease suffix
#   - Updates cross-module dependencies consistently
#
# =============================================================================

set -e

# Source common functions
readonly SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "${SCRIPT_DIR}/common.sh"

readonly BUMP_TYPE="${BUMP_TYPE:-prerelease}"
readonly DOCKER_IMAGE_SEMVER="mdelapenya/semver-tool:3.4.0"

MODULE="${1:-}"

if [[ -z "$MODULE" ]]; then
  echo "Usage: $0 <module>"
  exit 1
fi

LATEST_TAG=$(find_latest_tag "${MODULE}")
if [[ -z "$LATEST_TAG" ]]; then
  LATEST_TAG="${MODULE}/v0.1.0-alpha000"
fi

echo "Latest tag: ${LATEST_TAG}"

# Remove the module name from the latest tag
TAG_VERSION=$(echo "${LATEST_TAG}" | sed -E "s/^${MODULE}\///")
echo "Tag version: ${TAG_VERSION}"

# Strip leading zeros from the version before passing to semver-tool
CLEAN_TAG_VERSION=$(echo "${TAG_VERSION}" | sed -E 's/alpha0+([0-9]+)/alpha\1/')
echo "Clean tag version for semver-tool: ${CLEAN_TAG_VERSION}"

# Get the version to bump to from the semver-tool and the bump type
echo "Bumping ${BUMP_TYPE} version of ${CLEAN_TAG_VERSION}"
BASE_VERSION=$(docker run --rm --platform=linux/amd64 "${DOCKER_IMAGE_SEMVER}" bump "${BUMP_TYPE}" "${CLEAN_TAG_VERSION}")
if [[ "${BASE_VERSION}" == "" ]]; then
  echo "Failed to bump the version. Please check the semver-tool image and the bump type."
  exit 1
fi

# We need to extract the version from the output and add the leading zeros, in this case 002
# E.g.
# docker run --rm --platform=linux/amd64 -i mdelapenya/semver-tool:3.4.0 bump prerel 0.1.0-alpha001    
# 0.1.0-alpha2

# Extract the version from the output and add the leading zeros
# Extract base version and alpha number separately
BASE_PART=$(echo "${BASE_VERSION}" | sed -E "s/^([0-9]+\.[0-9]+\.[0-9]+)-alpha[0-9]+$/\1/")
ALPHA_NUM=$(echo "${BASE_VERSION}" | sed -E "s/^[0-9]+\.[0-9]+\.[0-9]+-alpha([0-9]+)$/\1/")

# When the alpha number and the base part are equal, it means that the version is a final version
# and we need to bump the base part. E.g. 1.0.0 has no alpha number.
if [[ "${ALPHA_NUM}" != "${BASE_PART}" ]]; then
  # Format with leading zeros
  NEXT_VERSION="${BASE_PART}-alpha$(printf %03d ${ALPHA_NUM})"
else
  NEXT_VERSION="${BASE_PART}"
fi

echo "Next version: ${NEXT_VERSION}"

# lowercase the module
MODULE=$(echo "$MODULE" | tr '[:upper:]' '[:lower:]')

# Update the version.go file with the next version

# The scripts are executed in the root of the repository, so no need to go to the module directory
if [[ "$DRY_RUN" == "true" ]]; then
  echo "[DRY RUN] Would update ${ROOT_DIR}/${MODULE}/version.go with version: ${NEXT_VERSION}"
else
  portable_sed "s/version = \"[^\"]*\"/version = \"${NEXT_VERSION}\"/" "${ROOT_DIR}/${MODULE}/version.go"
fi

# if next version does not start with v, add it
NEXT_vVERSION="${NEXT_VERSION}"
if [[ ! "${NEXT_VERSION}" =~ ^v ]]; then
  NEXT_vVERSION="v${NEXT_vVERSION}"
fi

NEXT_TAG="${MODULE}/${NEXT_vVERSION}"

echo "Next tag: ${NEXT_TAG}"

# Replace the entire line for the module in the go.mod files:
# github.com/docker/go-sdk/docker-client/v2.0.0-alpha001
# with
# github.com/docker/go-sdk/docker-client/v2.0.0-alpha002

# Find the line that contains the module and replace it with the new version

MODULES=$(go work edit -json | jq -r '.Use[] | "\(.DiskPath | ltrimstr("./"))"' | tr '\n' ' ' && echo)

# Save the next tag for the module to a file so that the release script can use it
execute_or_echo echo "${NEXT_vVERSION}" > "${ROOT_DIR}/.github/scripts/.${MODULE}-next-tag"

for m in $MODULES; do
  if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY RUN] Would update ${ROOT_DIR}/${m}/go.mod: ${GITHUB_REPO}/${MODULE} v${NEXT_VERSION}"
  else
    portable_sed "s|${GITHUB_REPO}/${MODULE} v[^[:space:]]*|${GITHUB_REPO}/${MODULE} v${NEXT_VERSION}|g" "${ROOT_DIR}/${m}/go.mod"
    # Update the go.sum file
    (cd "${ROOT_DIR}/${m}" && execute_or_echo go mod tidy)
  fi
done
