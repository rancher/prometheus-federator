#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

kubectl delete -f ./examples/helm-project-operator/ci/project-helm-chart.yaml
if kubectl get -n cattle-helm-system job/helm-delete-project-operator-example-chart-dummy --ignore-not-found; then
    if ! kubectl wait --for=condition=complete --timeout="${KUBECTL_WAIT_TIMEOUT}" -n cattle-helm-system job/helm-delete-project-operator-example-chart-dummy; then
        echo "ERROR: Helm Uninstall Job for Example Chart never completed after ${KUBECTL_WAIT_TIMEOUT}"
        kubectl logs job/helm-delete-project-operator-example-chart-dummy -n cattle-helm-system
        exit 1
    fi
fi

echo "PASS: Removing ProjectHelmChart successfully uninstalled Example Chart"