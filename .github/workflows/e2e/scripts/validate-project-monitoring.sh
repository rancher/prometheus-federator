#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

if ! kubectl -n cattle-project-p-example-monitoring rollout status statefulset alertmanager-cattle-project-p-example-m-alertmanager --timeout="${KUBECTL_WAIT_TIMEOUT}"; then
    echo "ERROR: Project Alertmanager did not roll out"
    exit 1;
fi

if ! kubectl -n cattle-project-p-example-monitoring rollout status statefulset prometheus-cattle-project-p-example-m-prometheus --timeout="${KUBECTL_WAIT_TIMEOUT}"; then
    echo "ERROR: Project Prometheus did not roll out"
    exit 1;
fi

if ! kubectl -n cattle-project-p-example-monitoring rollout status deployment cattle-project-p-example-monitoring-grafana --timeout="${KUBECTL_WAIT_TIMEOUT}"; then
    echo "ERROR: Project Grafana did not roll out"
    exit 1
fi

echo "PASS: Project Monitoring Stack is up and running"
