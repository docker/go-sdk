#!/bin/bash

# =============================================================================
# Common Shell Functions and Constants
# =============================================================================
# Description: Shared utilities and constants for Docker Go SDK release scripts
# 
# Environment Variables:
#   DRY_RUN          - Enable dry run mode (default: true)
#                      When true, commands are echoed instead of executed
#
# Functions:
#   curlGolangProxy  - Trigger Go proxy to fetch module (for publishing)
#   execute_or_echo  - Execute command or echo based on DRY_RUN setting
#   find_latest_tag  - Find latest tag for a given module
#   get_modules      - Get list of modules from go.work file
#   get_script_dir   - Get directory of the calling script
#   portable_sed     - Portable in-place sed editing
#
# Constants:
#   ROOT_DIR         - Root directory of the repository
#   CURRENT_DIR      - Current directory of the script
#   GITHUB_REPO      - GitHub repository identifier
#   DRY_RUN          - Dry run mode flag
#
# Usage:
#   source .github/scripts/common.sh
#
# =============================================================================

# Common constants and functions for release scripts

# Get the directory of the script that sources this file
get_script_dir() {
  cd "$( dirname "${BASH_SOURCE[1]}" )" && pwd
}

readonly CURRENT_DIR="$(get_script_dir)"
readonly ROOT_DIR="$(dirname $(dirname "${CURRENT_DIR}"))"
readonly BUILD_DIR="${ROOT_DIR}/.github/scripts/.build"
readonly GITHUB_REPO="github.com/docker/go-sdk"
readonly EXPECTED_ORIGIN_SSH="git@github.com:docker/go-sdk.git"
readonly EXPECTED_ORIGIN_HTTPS="https://${GITHUB_REPO}.git"
readonly DRY_RUN="${DRY_RUN:-true}"

# This function is used to trigger the Go proxy to fetch the module.
# See https://pkg.go.dev/about#adding-a-package for more details.
function curlGolangProxy() {
  local module="${1}"
  local module_version="${2}"

  # e.g.:
  #   github.com/docker/go-sdk/client/v0.1.0-alpha001.info
  #   github.com/docker/go-sdk/client/v0.0.1.info
  #   github.com/docker/go-sdk/client/v0.1.0.info
  #   github.com/docker/go-sdk/client/v1.0.0.info
  local module_url="https://proxy.golang.org/${GITHUB_REPO}/${module}/@v/${module_version}.info"

  execute_or_echo curl "${module_url}"
}


# Function to execute or echo commands based on DRY_RUN
execute_or_echo() {
  if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY RUN] Would execute: $*"
  else
    "$@"
  fi
}

# Validate that git remote origin points to the correct repository
# This prevents accidentally pushing to the wrong remote
validate_git_remote() {
  local actual_origin="$(git -C "${ROOT_DIR}" remote get-url origin 2>/dev/null || echo "")"

  if [[ -z "$actual_origin" ]]; then
    echo "❌ Error: No 'origin' remote found"
    echo "Please configure the origin remote first:"
    echo "  git remote add origin ${EXPECTED_ORIGIN_SSH}"
    exit 1
  fi

  # Accept both SSH and HTTPS formats for the docker/go-sdk repository
  if [[ "$actual_origin" != "$EXPECTED_ORIGIN_SSH" ]] && \
     [[ "$actual_origin" != "$EXPECTED_ORIGIN_HTTPS" ]]; then
    echo "❌ Error: Git remote 'origin' points to the wrong repository"
    echo "  Expected: ${EXPECTED_ORIGIN_SSH}"
    echo "            (or ${EXPECTED_ORIGIN_HTTPS})"
    echo "  Actual:   ${actual_origin}"
    echo ""
    echo "To fix this, update your origin remote:"
    echo "  git remote set-url origin ${EXPECTED_ORIGIN_SSH}"
    exit 1
  fi

  echo "✅ Git remote validation passed: origin → ${actual_origin}"
}

# Function to get modules from go.work
get_modules() {
  go work edit -json | jq -r '.Use[] | "\(.DiskPath | ltrimstr("./"))"' | tr '\n' ' ' && echo
}

# Function to find latest tag for a module
find_latest_tag() {
  local module="$1"
  git tag --list | grep -E "${module}/v[0-9]+\.[0-9]+\.[0-9]+.*" | sort -V | tail -n 1
}

# Function to get the next tag for a module
get_next_tag() {
  local module="$1"
  local next_tag_path="${BUILD_DIR}/${module}-next-tag"
  echo "${next_tag_path}"
}

# Extract version string from a version.go file
# Usage: get_version_from_file <path_to_version.go>
# Returns: version string (e.g., "0.1.0-alpha011")
get_version_from_file() {
  local file="$1"
  # Use pattern that allows arbitrary whitespace around = sign
  grep -o 'version[[:space:]]*=[[:space:]]*"[^"]*"' "$file" | cut -d'"' -f2
}

# Portable in-place sed editing that works on both BSD (macOS) and GNU (Linux) sed
portable_sed() {
  local pattern="$1"
  local file="$2"

  # Detect sed version and use appropriate syntax
  if sed --version >/dev/null 2>&1; then
    # GNU sed (Linux)
    execute_or_echo sed -i "$pattern" "$file"
  else
    # BSD sed (macOS)
    execute_or_echo sed -i '' "$pattern" "$file"
  fi
}
