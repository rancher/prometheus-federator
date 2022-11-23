#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

if ! kubectl -n cattle-monitoring-system rollout status deployment prometheus-federator --timeout="${KUBECTL_WAIT_TIMEOUT}"; then 
    echo "ERROR: Prometheus Federator did not roll out";
    kubectl get pods -n cattle-monitoring-system -o yaml;
    exit 1;
fi;

echo "PASS: Prometheus Federator is up and running"