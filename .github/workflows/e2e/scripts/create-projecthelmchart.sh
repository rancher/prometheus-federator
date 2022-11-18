#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

if [[ "${E2E_CI}" == "true" ]]; then
    kubectl apply -f ./examples/ci-example.yaml
else
    kubectl apply -f ./examples/example.yaml
fi
sleep ${DEFAULT_SLEEP_TIMEOUT_SECONDS};
if ! kubectl wait --for=condition=complete --timeout="${KUBECTL_WAIT_TIMEOUT}" -n cattle-monitoring-system job/helm-install-cattle-project-p-example-monitoring; then
    echo "ERROR: Helm Install Job for Project Monitoring Stack never completed after ${KUBECTL_WAIT_TIMEOUT} seconds"
    kubectl logs job/helm-install-cattle-project-p-example-monitoring -n cattle-monitoring-system
    exit 1
fi
kubectl logs job/helm-install-cattle-project-p-example-monitoring -n cattle-monitoring-system

echo "PASS: Adding ProjectHelmChart successfully installed Project Prometheus Stack"
