#!/bin/bash
set -e

source $(dirname $0)/entry
source $(dirname $0)/cluster-args.sh

cd $(dirname $0)/../../../..

latest_chart=./packages/helm-project-operator/charts

helm upgrade --install --create-namespace -n cattle-helm-system helm-project-operator --set image.registry='',image.repository=${REPO:-rancher}/helm-project-operator,image.tag=${TAG:-dev} ${cluster_args} ${RANCHER_HELM_ARGS} ${latest_chart}

echo "PASS: Helm Project Operator has been installed"
