#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

HELM_REPO="rancher-charts"

cd $(dirname $0)/../../../..

helm version

helm repo add ${HELM_REPO} https://charts.rancher.io
helm repo update

echo "Create required \`cattle-fleet-system\` namespace"
kubectl create namespace cattle-fleet-system

echo "Installing rancher monitoring crd with :\n"

helm search repo ${HELM_REPO}/rancher-monitoring-crd --versions --max-col-width=0 | head -n 2

helm upgrade --install --create-namespace -n cattle-monitoring-system rancher-monitoring-crd ${HELM_REPO}/rancher-monitoring-crd

if [[ "${E2E_CI}" == "true" ]]; then
    e2e_args="--set grafana.resources=null --set prometheus.prometheusSpec.resources=null --set alertmanager.alertmanagerSpec.resources=null"
fi

case "${KUBERNETES_DISTRIBUTION_TYPE}" in
"k3s")
    cluster_args="--set k3sServer.enabled=true"
    ;;
"rke")
    cluster_args="--set rkeControllerManager.enabled=true --set rkeScheduler.enabled=true --set rkeProxy.enabled=true --set rkeEtcd.enabled=true"
    ;;
"rke2")
    cluster_args="--set rke2ControllerManager.enabled=true --set rke2Scheduler.enabled=true --set rke2Proxy.enabled=true --set rke2Etcd.enabled=true"
    ;;
*)
    echo "KUBERNETES_DISTRIBUTION_TYPE=${KUBERNETES_DISTRIBUTION_TYPE} is unknown"
    exit 1
esac

echo "Installing rancher monitoring with :\n"

helm search repo ${HELM_REPO}/rancher-monitoring --versions --max-col-width=0 | head -n 2
helm upgrade --install --create-namespace -n cattle-monitoring-system rancher-monitoring ${cluster_args} ${e2e_args} ${RANCHER_HELM_ARGS} ${HELM_REPO}/rancher-monitoring

echo "PASS: Rancher Monitoring has been installed"
