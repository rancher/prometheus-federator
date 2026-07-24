#!/bin/bash
set -e
set -x

source $(dirname $0)/entry
source $(dirname $0)/cluster-args.sh

cd $(dirname $0)/../../../..
source "$(pwd)/scripts/util-team-charts"

make package-helm

helm upgrade --install ${HELM_EXTRA_FLAGS} --create-namespace -n cattle-monitoring-system prometheus-federator \
  --set helmProjectOperator.image.repository=${REPO:-rancher}/prometheus-federator \
  --set helmProjectOperator.image.tag=${TAG:-dev} \
  --set helmProjectOperator.chartSource.name='rancher-project-monitoring' \
  --set helmProjectOperator.chartSource.repo='https://raw.githubusercontent.com/rancher/ob-team-charts/refs/heads/main' \
  --set helmProjectOperator.chartSource.version='1.0.0' \
  --set helmProjectOperator.valuesOverride.grafana.image.repository='rancher/appco-grafana' \
  --set helmProjectOperator.valuesOverride.grafana.image.tag='12.3.1-1.12' \
  --set helmProjectOperator.valuesOverride.grafana.downloadDashboardsImage.repository='rancher/appco-curl' \
  --set helmProjectOperator.valuesOverride.grafana.downloadDashboardsImage.tag='8.14.1-7.1' \
  --set helmProjectOperator.valuesOverride.grafana.initChownData.image.repository='rancher/mirrored-library-busybox' \
  --set helmProjectOperator.valuesOverride.grafana.initChownData.image.tag='1.31.1' \
  --set helmProjectOperator.valuesOverride.grafana.sidecar.image.repository='rancher/appco-k8s-sidecar' \
  --set helmProjectOperator.valuesOverride.grafana.sidecar.image.tag='2.1.2-1.10' \
  --set helmProjectOperator.valuesOverride.grafana.imageRenderer.image.repository='rancher/mirrored-grafana-grafana-image-renderer' \
  --set helmProjectOperator.valuesOverride.grafana.imageRenderer.image.tag='v5.1.0' \
  ${cluster_args} \
  ${RANCHER_HELM_ARGS} ./build/charts/prometheus-federator

echo "PASS: Prometheus Federator has been installed"
