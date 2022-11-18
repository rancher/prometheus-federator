#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

helm repo add rancher-charts https://charts.rancher.io
helm repo update
helm upgrade --install --create-namespace -n cattle-monitoring-system rancher-monitoring-crd rancher-charts/rancher-monitoring-crd

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

helm upgrade --install --create-namespace -n cattle-monitoring-system rancher-monitoring ${cluster_args} ${e2e_args} ${RANCHER_HELM_ARGS} rancher-charts/rancher-monitoring

echo "PASS: Rancher Monitoring has been installed"
