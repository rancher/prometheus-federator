#!/usr/bin/env bash
# Update packages/rancher-monitoring/prometheus-federator/package.yaml in rancher/charts for a new release.
#
# Inputs (env):
#   TAG         - prometheus-federator release tag (e.g. v7.0.1-rc.2) (required)
#   CHARTS_DIR  - path to rancher/charts clone (required)
#   PACKAGE     - package path under packages/ (set by common.sh)
#   CHART_NAME  - chart name (set by common.sh: "prometheus-federator")
#
# Updates:
#   url     → new chart tarball URL for TAG
#   version → auto-incremented: major if base major changed, minor if base minor changed, else patch
#
# Output: git commit in CHARTS_DIR
set -euo pipefail
source "$(dirname "$0")/common.sh"

require_var TAG
require_charts_dir

PF_REPO_URL="https://github.com/rancher/prometheus-federator"
TAG_NO_V="${TAG#v}"  # e.g. 7.0.1-rc.2

PACKAGE_YAML="$CHARTS_DIR/packages/$PACKAGE/package.yaml"
if [ ! -f "$PACKAGE_YAML" ]; then
  echo "ERROR: package.yaml not found at $PACKAGE_YAML" >&2
  exit 1
fi

# --- Compute new URL ---
NEW_URL="${PF_REPO_URL}/releases/download/${TAG}/${CHART_NAME}-${TAG_NO_V}.tgz"

# --- Compute new charts version ---
# Read old prometheus-federator version from the current URL and strip any prerelease suffix
CURRENT_URL=$(yq e '.url' "$PACKAGE_YAML")
OLD_VERSION=$(echo "$CURRENT_URL" | sed "s|.*/${CHART_NAME}-||" | sed 's|\.tgz$||')
OLD_BASE=$(echo "$OLD_VERSION" | sed 's/-.*//')

# New base version (strip prerelease)
NEW_BASE=$(echo "$TAG_NO_V" | sed 's/-.*//')

CURRENT_CHARTS_VERSION=$(yq e '.version' "$PACKAGE_YAML")
CHARTS_MAJOR=$(echo "$CURRENT_CHARTS_VERSION" | cut -d. -f1)
CHARTS_MINOR=$(echo "$CURRENT_CHARTS_VERSION" | cut -d. -f2)
CHARTS_PATCH=$(echo "$CURRENT_CHARTS_VERSION" | cut -d. -f3)

OLD_MAJOR=$(echo "$OLD_BASE" | cut -d. -f1)
NEW_MAJOR=$(echo "$NEW_BASE" | cut -d. -f1)
OLD_MINOR=$(echo "$OLD_BASE" | cut -d. -f2)
NEW_MINOR=$(echo "$NEW_BASE" | cut -d. -f2)

if [ "$OLD_BASE" = "$NEW_BASE" ]; then
  # Same base version (RC→RC or RC→stable): version was already bumped, keep it
  NEW_CHARTS_VERSION="$CURRENT_CHARTS_VERSION"
elif [ "$NEW_MAJOR" != "$OLD_MAJOR" ]; then
  # Major version changed (new Rancher version line seeded from previous branch)
  NEW_CHARTS_VERSION="$((CHARTS_MAJOR + 1)).0.0"
elif [ "$NEW_MINOR" != "$OLD_MINOR" ]; then
  # Minor version changed
  NEW_CHARTS_VERSION="${CHARTS_MAJOR}.$((CHARTS_MINOR + 1)).0"
else
  # Patch version changed
  NEW_CHARTS_VERSION="${CHARTS_MAJOR}.${CHARTS_MINOR}.$((CHARTS_PATCH + 1))"
fi

# --- Apply updates ---
yq e -i ".url = \"$NEW_URL\"" "$PACKAGE_YAML"
yq e -i ".version = \"$NEW_CHARTS_VERSION\"" "$PACKAGE_YAML"

summary "  - prometheus-federator: \`$OLD_VERSION\` → \`$TAG_NO_V\`"
summary "  - Charts version: \`$CURRENT_CHARTS_VERSION\` → \`$NEW_CHARTS_VERSION\`"

commit_if_changed "chore(charts): Update $CHART_NAME package.yaml for $TAG"
