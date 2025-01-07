#!/bin/bash
set -e

source $(dirname $0)/entry
source $(dirname $0)/cluster-args.sh

cd $(dirname $0)/../../../..
source "$(pwd)/scripts/util-team-charts"

NEWEST_CHART_VERSION=$(newest-chart-version "helm-project-operator")
fetch-team-chart "helm-project-operator" "$NEWEST_CHART_VERSION"
LATEST_CHART_PATH="./build/charts/helm-project-operator-${NEWEST_CHART_VERSION}.tgz"
tar -xvzf "$LATEST_CHART_PATH" -C ./build/charts/

helm upgrade --install --create-namespace -n cattle-helm-system helm-project-operator --set image.registry='',image.repository=${REPO:-rancher}/helm-project-operator,image.tag=${TAG:-dev} ${cluster_args} ${RANCHER_HELM_ARGS} ./build/charts/helm-project-operator

echo "PASS: Helm Project Operator has been installed"
