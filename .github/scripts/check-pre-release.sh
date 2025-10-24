#!/bin/bash

# =============================================================================
# Pre-Release Check Script
# =============================================================================
# Description: Verifies that pre-release was completed successfully for a module
#              by checking if the next-tag file exists and matches version.go
#
# Usage: ./.github/scripts/check-pre-release.sh <module>
#
# Arguments:
#   module           - Name of the module to check (required)
#                      Examples: client, container, config, context, image, network
#
# Exit Codes:
#   0 - Check passed
#   1 - Check failed (missing files or version mismatch)
#
# Examples:
#   ./.github/scripts/check-pre-release.sh client
#   ./.github/scripts/check-pre-release.sh container
#
# Dependencies:
#   - grep (for parsing version.go)
#
# Files Checked:
#   - .github/scripts/.build/<module>-next-tag  - Pre-release version file
#   - <module>/version.go                       - Current version file
#
# =============================================================================

set -e

# Source common functions
readonly SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "${SCRIPT_DIR}/common.sh"

# Get module name from argument and lowercase it
readonly MODULE=$(echo "${1:-}" | tr '[:upper:]' '[:lower:]')

if [[ -z "$MODULE" ]]; then
  echo "Error: Module name is required"
  echo "Usage: $0 <module>"
  echo "Example: $0 client"
  exit 1
fi

echo "Checking if pre-release was completed for module: ${MODULE}"

# Check if next-tag file exists
readonly BUILD_FILE="${BUILD_DIR}/${MODULE}-next-tag"
if [[ ! -f "${BUILD_FILE}" ]]; then
  echo "Error: Missing build file for module '${MODULE}' at ${BUILD_FILE}"
  echo "Please run 'make pre-release-all' or 'make pre-release' first (with DRY_RUN=false)"
  exit 1
fi

# Read next version from build file
readonly NEXT_VERSION=$(cat "${BUILD_FILE}" | tr -d '\n')
readonly NEXT_VERSION_NO_V="${NEXT_VERSION#v}"

# Check if version.go exists
readonly VERSION_FILE="${ROOT_DIR}/${MODULE}/version.go"
if [[ ! -f "${VERSION_FILE}" ]]; then
  echo "Error: version.go not found at ${VERSION_FILE}"
  exit 1
fi

# Read current version from version.go
readonly CURRENT_VERSION=$(get_version_from_file "${VERSION_FILE}")

# Compare versions
if [[ "${CURRENT_VERSION}" != "${NEXT_VERSION_NO_V}" ]]; then
  echo "Error: Version mismatch for module '${MODULE}'"
  echo "  Expected (from ${BUILD_FILE}): ${NEXT_VERSION_NO_V}"
  echo "  Actual (from ${VERSION_FILE}): ${CURRENT_VERSION}"
  echo "Please run 'make pre-release-all' or 'make pre-release' again (with DRY_RUN=false)"
  exit 1
fi

echo "✅ Pre-release check passed for module: ${MODULE} (version: ${NEXT_VERSION_NO_V})"
exit 0
