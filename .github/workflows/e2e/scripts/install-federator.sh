#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

latest_chart=$(find ./charts/prometheus-federator -type d -maxdepth 1 -mindepth 1 | tr - \~ | sort -rV | tr \~ - | head -n1)

case "${KUBERNETES_DISTRIBUTION_TYPE}" in
"k3s")
    cluster_args="--set helmProjectOperator.helmController.enabled=false"
    ;;
"rke")
    cluster_args=""
    ;;
"rke2")
    cluster_args="--set helmProjectOperator.helmController.enabled=false"
    ;;
*)
    echo "KUBERNETES_DISTRIBUTION_TYPE=${KUBERNETES_DISTRIBUTION_TYPE} is unknown"
    exit 1
esac

helm upgrade --install --create-namespace -n cattle-monitoring-system prometheus-federator --set helmProjectOperator.image.repository=${REPO:-rancher}/prometheus-federator --set helmProjectOperator.image.tag=${TAG:-dev} ${cluster_args} ${RANCHER_HELM_ARGS} ${latest_chart}

echo "PASS: Prometheus Federator has been installed"
