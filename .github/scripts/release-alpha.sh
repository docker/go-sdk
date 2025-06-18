#!/bin/bash

set -e

BASE_VERSION="${1:-v0.1.0}"
MODULE="${2:-}"

if [[ -z "$MODULE" ]]; then
  echo "Usage: $0 <base-version> <module>"
  exit 1
fi

# lowercase the module
MODULE=$(echo "$MODULE" | tr '[:upper:]' '[:lower:]')

PREFIX="${BASE_VERSION}-alpha"

# Find the latest matching tag
LATEST_TAG=$(git tag --list | grep -E "${MODULE}/${PREFIX}[0-9][0-9][0-9]" | sort -V | tail -n 1)

echo "Latest tag: ${LATEST_TAG}"

if [[ -z "$LATEST_TAG" ]]; then
  NEXT_NUMBER="001"
else
  LAST_NUMBER=$(echo "$LATEST_TAG" | sed -E "s/^${MODULE}\/${PREFIX}([0-9]{3})$/\1/")
  NEXT_NUMBER=$(printf "%03d" $((10#$LAST_NUMBER + 1)))
fi

NEXT_TAG="${MODULE}/${PREFIX}${NEXT_NUMBER}"

# Output the next tag
echo "Next alpha tag: ${NEXT_TAG}"

# Optional: create the tag
read -p "Do you want to create and push this tag? [y/N]: " confirm
if [[ "$confirm" =~ ^[Yy]$ ]]; then
  git tag "$NEXT_TAG"
  git push origin "$NEXT_TAG"
fi
