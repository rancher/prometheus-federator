#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

if [[ "${E2E_CI}" == "true" ]]; then
    kubectl delete -f ./examples/prometheus-federator/ci/project-helm-chart.yaml
else
    kubectl delete -f ./examples/prometheus-federator/project-helm-chart.yaml
fi

JOB_NAME=helm-delete-cattle-project-p-example-monitoring

if kubectl get -n cattle-monitoring-system job "${JOB_NAME}" >/dev/null 2>&1; then
    kubectl wait --for=condition=complete --timeout="${KUBECTL_WAIT_TIMEOUT}" -n cattle-monitoring-system job/"${JOB_NAME}" || {
        echo "ERROR: Helm Uninstall Job for Project Monitoring Stack never completed after ${KUBECTL_WAIT_TIMEOUT}"
        kubectl logs job/"${JOB_NAME}" -n cattle-monitoring-system || true
        exit 1
    }
fi

if [[ $(kubectl get -n cattle-project-p-example -l "release=cattle-project-p-example-monitoring" secrets -o jsonpath='{.items[].metadata.name}' --ignore-not-found) != "cattle-project-p-example-m-alertmanager-secret" ]]; then
    echo "ERROR: Expected Project Alertmanager Secret to be left behind in the namespace"
    exit 1
fi

if [[ -n $(kubectl get -n cattle-project-p-example -l "release=cattle-project-p-example-monitoring" pods -o jsonpath='{.items[].metadata.name}' --ignore-not-found) ]]; then
    echo "ERROR: Expected all pods of the Helm Release to be cleaned up on deleting the ProjectHelmChart"
    exit 1
fi

echo "PASS: Removing ProjectHelmChart successfully uninstalled Project Prometheus Stack"
