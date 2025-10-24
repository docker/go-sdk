#!/bin/bash

# =============================================================================
# Go Proxy Refresh Script
# =============================================================================
# Description: Triggers the Go proxy to refresh/fetch a module version
#              This is useful to ensure pkg.go.dev has the latest version
#
# Usage: ./.github/scripts/refresh-proxy.sh <module>
#
# Arguments:
#   module           - Name of the module to refresh (required)
#                      Examples: client, container, config, context, image, network
#
# Examples:
#   ./.github/scripts/refresh-proxy.sh client
#   ./.github/scripts/refresh-proxy.sh container
#
# Dependencies:
#   - git (for finding latest tag)
#   - curl (for triggering Go proxy)
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

echo "Refreshing Go proxy for module: ${MODULE}"

# Check if version.go exists
readonly VERSION_FILE="${ROOT_DIR}/${MODULE}/version.go"
if [[ ! -f "${VERSION_FILE}" ]]; then
  echo "Error: version.go not found at ${VERSION_FILE}"
  exit 1
fi

# Read version from version.go
VERSION=$(get_version_from_file "${VERSION_FILE}")

if [[ -z "$VERSION" ]]; then
  echo "Error: Could not extract version from ${VERSION_FILE}"
  exit 1
fi

# Ensure version has v prefix for the tag
if [[ ! "${VERSION}" =~ ^v ]]; then
  VERSION="v${VERSION}"
fi

echo "Current version: ${VERSION}"
echo "Triggering Go proxy refresh..."

# Trigger Go proxy (bypass dry-run since this is a read-only operation)
DRY_RUN=false curlGolangProxy "${MODULE}" "${VERSION}"

echo "✅ Go proxy refresh completed for ${MODULE}@${VERSION}"
echo "The module should be available at: https://pkg.go.dev/${GITHUB_REPO}/${MODULE}@${VERSION}"
