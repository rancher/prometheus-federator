#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

helm repo add rancher-charts https://charts.rancher.io;
helm repo update;
helm upgrade --install --wait --create-namespace -n cattle-monitoring-system rancher-monitoring-crd rancher-charts/rancher-monitoring-crd;
helm upgrade --install --wait --create-namespace -n cattle-monitoring-system rancher-monitoring --set k3sServer.enabled=true --set grafana.resources=null --set prometheus.prometheusSpec.resources=null --set alertmanager.alertmanagerSpec.resources=null rancher-charts/rancher-monitoring;

echo "PASS: Rancher Monitoring has been installed"