#!/usr/bin/env bash
# Prepares the chart of a release for the ship-family-staged action of
# rancher/charts: copies prometheus-federator from a checkout of
# the release tag to <output dir>/prometheus-federator, and fills in the copy with
# what the tag gives, the way scripts/package-helm does for the GitHub release. The
# action stamps, checks, packages and pushes the copy, so the chart in the
# repository stays as it is, and the push-to-charts workflow keeps working with it.
#
# What it sets in the copy: appVersion and catalog.cattle.io/upstream-version, the
# tag of the image in values.yaml, and the bounds of the rancher-version and
# kube-version annotations.
#
# A bound that is already there is kept. When only the lower bound is written, the
# upper bound is added: the next minor of Rancher, and three minors after the
# lower bound of Kubernetes, which is the window of Kubernetes versions that
# Rancher supports. The chart version is not set here: the action works it out
# from the appVersion and the rancher-version annotation of the main chart.
#
# Usage: prepare-charts.sh <tag> <source dir> <output dir>
#   e.g. prepare-charts.sh v12.0.0-rc.4 release "$RUNNER_TEMP/charts"
#
# <source dir> is a checkout of the tag. This script is run from another one, so a
# tag that is older than the script can be released.
set -euo pipefail

TAG=${1:?usage: $0 <tag> <source dir> <output dir>}
SOURCE_DIR=${2:?usage: $0 <tag> <source dir> <output dir>}
OUT_DIR=${3:?usage: $0 <tag> <source dir> <output dir>}

if ! [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]; then
  echo "Error: tag $TAG must look like vX.Y.Z or vX.Y.Z-rc.N" >&2
  exit 1
fi
UPSTREAM_VERSION="${TAG#v}"

if [ ! -f "$SOURCE_DIR/charts/prometheus-federator/Chart.yaml" ]; then
  echo "Error: there is no Chart.yaml in $SOURCE_DIR/charts/prometheus-federator" >&2
  exit 1
fi

# bound_annotation <Chart.yaml> <annotation> <minors>: when the annotation is
# written as ">= X.Y.Z-0", with no upper bound, adds "< X.(Y+minors).0-0".
bound_annotation() {
  local file=$1 name=$2 minors=$3 value major minor
  value=$(sed -n "s|^ *catalog.cattle.io/${name}: *['\"]\{0,1\}\(.*[^'\"]\)['\"]\{0,1\} *\$|\1|p" "$file")
  if [[ "$value" =~ ^\>=\ *([0-9]+)\.([0-9]+)\.[0-9]+-0$ ]]; then
    major="${BASH_REMATCH[1]}"
    minor="${BASH_REMATCH[2]}"
    sed -i "s|\(catalog.cattle.io/${name}:\).*|\1 '${value} < ${major}.$((minor + minors)).0-0'|" "$file"
  fi
}

CHART=prometheus-federator

mkdir -p "$OUT_DIR"
rm -rf "${OUT_DIR:?}/$CHART"
cp -r "$SOURCE_DIR/charts/$CHART" "$OUT_DIR/$CHART"

chart_yaml="$OUT_DIR/$CHART/Chart.yaml"
sed -i "s|^appVersion:.*|appVersion: \"${TAG}\"|" "$chart_yaml"
sed -i "s|\(catalog.cattle.io/upstream-version:\).*|\1 ${UPSTREAM_VERSION}|" "$chart_yaml"
bound_annotation "$chart_yaml" rancher-version 1
bound_annotation "$chart_yaml" kube-version 3

values_yaml="$OUT_DIR/$CHART/values.yaml"
sed -i "s|tag: ''|tag: ${TAG}|" "$values_yaml"
if grep -q "tag: ''" "$values_yaml"; then
  echo "Error: the image tag placeholder is left in $values_yaml" >&2
  exit 1
fi

echo "Prepared $CHART for $TAG in $OUT_DIR" >&2
