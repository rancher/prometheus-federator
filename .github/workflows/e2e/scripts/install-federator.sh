#!/bin/bash
set -e
set -x

source $(dirname $0)/entry
source $(dirname $0)/cluster-args.sh

cd $(dirname $0)/../../../..
source "$(pwd)/scripts/util-team-charts"

make package-helm

helm upgrade --install --create-namespace -n cattle-monitoring-system prometheus-federator --set helmProjectOperator.image.repository=${REPO:-rancher}/prometheus-federator --set helmProjectOperator.image.tag=${TAG:-dev} ${cluster_args} ${RANCHER_HELM_ARGS} ./build/charts/prometheus-federator

echo "PASS: Prometheus Federator has been installed"
