#!/bin/bash
set -e
set -x

source $(dirname $0)/entry
source $(dirname $0)/cluster-args.sh

cd $(dirname $0)/../../../..

latest_chart=$(find ./charts/prometheus-federator -type d -maxdepth 1 -mindepth 1 | tr - \~ | sort -rV | tr \~ - | head -n1)

helm upgrade --install --create-namespace -n cattle-monitoring-system prometheus-federator --set helmProjectOperator.image.repository=${REPO:-rancher}/prometheus-federator --set helmProjectOperator.image.tag=${TAG:-dev} ${cluster_args} ${RANCHER_HELM_ARGS} ${latest_chart}

echo "PASS: Prometheus Federator has been installed"
