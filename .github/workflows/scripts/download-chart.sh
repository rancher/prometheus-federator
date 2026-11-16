#!/usr/bin/env bash
# Downloads the chart archive of a release and extracts it for the
# ship-family-staged action of rancher/charts: scripts/package-helm already
# stamped appVersion, version and the image tag in it when the release was built,
# so this only fetches what was published, without touching its contents.
#
# Usage: download-chart.sh <tag> <output dir>
#   e.g. download-chart.sh v8.0.0-rc.1 "$RUNNER_TEMP/charts"
#
# Environment:
#   GH_TOKEN           can read the releases of the repository
#   GITHUB_REPOSITORY  the repository, such as rancher/prometheus-federator
#
# Requires gh and tar on PATH.
set -euo pipefail

TAG=${1:?usage: $0 <tag> <output dir>}
OUT_DIR=${2:?usage: $0 <tag> <output dir>}
REPOSITORY=${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is missing}

if ! [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]; then
  echo "Error: tag $TAG must look like vX.Y.Z or vX.Y.Z-rc.N" >&2
  exit 1
fi
VERSION="${TAG#v}"
ARCHIVE="prometheus-federator-${VERSION}.tgz"

mkdir -p "$OUT_DIR"
rm -rf "${OUT_DIR:?}/prometheus-federator" "${OUT_DIR:?}/${ARCHIVE}"
gh release download "$TAG" --repo "$REPOSITORY" --pattern "$ARCHIVE" --dir "$OUT_DIR"
tar -xzf "$OUT_DIR/$ARCHIVE" -C "$OUT_DIR"
rm "$OUT_DIR/$ARCHIVE"

echo "Downloaded and extracted $ARCHIVE to $OUT_DIR" >&2
