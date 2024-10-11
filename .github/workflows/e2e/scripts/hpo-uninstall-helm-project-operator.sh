#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

helm uninstall --wait -n cattle-helm-system helm-project-operator

echo "PASS: Helm Project Operator has been uninstalled"
