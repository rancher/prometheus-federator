#!/usr/bin/env bash
# Remove the previous RC charts version when publishing a new RC or stable for the same base version.
# Skipped automatically if the old prometheus-federator base version differs from the new one (no RC to clean up).
#
# Prerequisites: none (runs before update-package-yaml.sh)
#
# Inputs (env):
#   TAG        - new prometheus-federator release tag (e.g. v7.0.1-rc.2 or v7.0.1) (required)
#   CHARTS_DIR - path to rancher/charts clone (required)
#   PACKAGE    - package path under packages/ (set by common.sh)
#   CHART_NAME - chart name (set by common.sh: "prometheus-federator")
#
# Output: git commit in CHARTS_DIR (only if removal was performed)
set -euo pipefail
source "$(dirname "$0")/common.sh"

require_var TAG
require_charts_dir

PACKAGE_YAML="$CHARTS_DIR/packages/$PACKAGE/package.yaml"

# Extract old prometheus-federator version from current URL and strip any prerelease suffix
CURRENT_URL=$(yq e '.url' "$PACKAGE_YAML")
OLD_VERSION=$(echo "$CURRENT_URL" | sed "s|.*/${CHART_NAME}-||" | sed 's|\.tgz$||')
OLD_BASE=$(echo "$OLD_VERSION" | sed 's/-.*//')

# New base version (strip prerelease)
TAG_NO_V="${TAG#v}"
NEW_BASE=$(echo "$TAG_NO_V" | sed 's/-.*//')

# Only remove if the base version is unchanged AND the old version was a prerelease
if [ "$OLD_BASE" != "$NEW_BASE" ]; then
  summary "  - Base version changed ($OLD_BASE → $NEW_BASE), no RC to remove"
  exit 0
fi

if [[ "$OLD_VERSION" != *"-"* ]]; then
  summary "  - Old version $OLD_VERSION is not a prerelease, nothing to remove"
  exit 0
fi

PREV_CHARTS_BASE_VERSION=$(yq e '.version' "$PACKAGE_YAML")
PREV_CHARTS_VERSION="${PREV_CHARTS_BASE_VERSION}+up${OLD_VERSION}"
summary "  - Removing previous RC: $CHART_NAME $PREV_CHARTS_VERSION"

make -C "$CHARTS_DIR" remove CHART="$CHART_NAME" VERSION="$PREV_CHARTS_VERSION" || true

# Also remove from release.yaml (make remove does not touch it)
RELEASE_YAML="$CHARTS_DIR/release.yaml"
yq e -i ".${CHART_NAME} |= map(select(. != \"${PREV_CHARTS_VERSION}\"))" "$RELEASE_YAML"

commit_if_changed "chore(charts): Remove previous RC $CHART_NAME $PREV_CHARTS_VERSION"
