#!/usr/bin/env bash
# Prepend the new combined version to prometheus-federator in release.yaml.
#
# Prerequisites: generate-assets.sh
#
# Inputs (env):
#   TAG        - prometheus-federator release tag (e.g. v7.0.1-rc.2) (required)
#   CHARTS_DIR - path to rancher/charts clone (required)
#   PACKAGE    - package path under packages/ (set by common.sh)
#   CHART_NAME - chart name (set by common.sh: "prometheus-federator")
set -euo pipefail
source "$(dirname "$0")/common.sh"

require_var TAG
require_charts_dir

TAG_NO_V="${TAG#v}"
CHARTS_VERSION=$(yq e '.version' "$CHARTS_DIR/packages/$PACKAGE/package.yaml")
COMBINED_VERSION="${CHARTS_VERSION}+up${TAG_NO_V}"

RELEASE_YAML="$CHARTS_DIR/release.yaml"

yq e -i ".${CHART_NAME} |= [\"${COMBINED_VERSION}\"] + ." "$RELEASE_YAML"
summary "  - Added \`${CHART_NAME}\`: \`${COMBINED_VERSION}\`"

if ! git -C "$CHARTS_DIR" diff --quiet --exit-code -- release.yaml; then
  git -C "$CHARTS_DIR" add release.yaml
  git -C "$CHARTS_DIR" commit -m "chore(charts): Update release.yaml for $CHART_NAME ${COMBINED_VERSION}"
fi
