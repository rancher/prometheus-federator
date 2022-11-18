#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

latest_chart=$(find ./charts/prometheus-federator -type d -maxdepth 1 -mindepth 1 | tr - \~ | sort -rV | tr \~ - | head -n1)
helm upgrade --install --wait --create-namespace -n cattle-monitoring-system prometheus-federator --set helmProjectOperator.image.tag=dev --set helmProjectOperator.helmController.enabled=false ${latest_chart};

echo "PASS: Prometheus Federator has been installed"