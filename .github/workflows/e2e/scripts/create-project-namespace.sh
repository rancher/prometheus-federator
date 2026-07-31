#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

USE_RANCHER=${USE_RANCHER:-"false"}
if [ "$USE_RANCHER" = "true" ]; then
  kubectl apply -f ./examples/prometheus-federator/ci/project.yaml
fi

kubectl apply -f ./examples/prometheus-federator/ci/namespace.yaml

sleep "${DEFAULT_SLEEP_TIMEOUT_SECONDS}"
if ! kubectl get namespace cattle-project-p-example; then
    echo "ERROR: Expected cattle-project-p-example namespace to exist after ${DEFAULT_SLEEP_TIMEOUT_SECONDS} seconds, not found"
    exit 1
fi

echo "PASS: Project Registration Namespace was created"
