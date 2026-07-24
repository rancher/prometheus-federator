#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

kubectl apply -f ./examples/helm-project-operator/ci/project-helm-chart.yaml
sleep ${DEFAULT_SLEEP_TIMEOUT_SECONDS};

if ! kubectl get -n cattle-helm-system job/helm-install-project-operator-example-chart-dummy; then
    echo "ERROR: Helm Install Job for Example Chart was never created after ${KUBECTL_WAIT_TIMEOUT} seconds"
    echo "PROJECT HELM CHARTS:"
    kubectl get projecthelmchart -n cattle-project-p-example -o yaml
    echo "HELM CHARTS:"
    kubectl get helmcharts -n cattle-helm-system -o yaml
    echo "HELM RELEASES:"
    kubectl get helmreleases -n cattle-helm-system -o yaml
    echo "HELM PROJECT OPERATOR:"
    kubectl logs deployment/helm-project-operator -n cattle-helm-system
    exit 1
fi

if ! kubectl wait --for=condition=complete --timeout="${KUBECTL_WAIT_TIMEOUT}" -n cattle-helm-system job/helm-install-project-operator-example-chart-dummy; then
    echo "ERROR: Helm Install Job for Example Chart never completed after ${KUBECTL_WAIT_TIMEOUT} seconds"
    kubectl logs job/helm-install-project-operator-example-chart-dummy -n cattle-helm-system
    exit 1
fi
kubectl logs job/helm-install-project-operator-example-chart-dummy -n cattle-helm-system

echo "PASS: Adding ProjectHelmChart successfully installed Example Chart"
