#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

if ! kubectl -n cattle-helm-system rollout status deployment helm-project-operator --timeout="${KUBECTL_WAIT_TIMEOUT}"; then 
    echo "ERROR: Helm Project Operator did not roll out"
    kubectl get pods -n cattle-helm-system -o yaml
    exit 1
fi

echo "PASS: Helm Project Operator is up and running"
