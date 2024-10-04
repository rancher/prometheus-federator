#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

kubectl create namespace e2e-prometheus-federator || true
kubectl label namespace e2e-prometheus-federator field.cattle.io/projectId=p-example --overwrite
kubectl annotate namespace e2e-prometheus-federator field.cattle.io/projectId=local:p-example --overwrite
sleep "${DEFAULT_SLEEP_TIMEOUT_SECONDS}"
if ! kubectl get namespace cattle-project-p-example; then
    echo "ERROR: Expected cattle-project-p-example namespace to exist after ${DEFAULT_SLEEP_TIMEOUT_SECONDS} seconds, not found"
    exit 1
fi

echo "PASS: Project Registration Namespace was created"
