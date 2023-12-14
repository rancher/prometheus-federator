#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

helm uninstall --wait -n cattle-monitoring-system prometheus-federator

echo "PASS: Prometheus Federator has been uninstalled"
