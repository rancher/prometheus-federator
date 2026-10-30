#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

case "${KUBERNETES_DISTRIBUTION_TYPE}" in
"k3s")
    cluster_args=""
    ;;
"rke")
    cluster_args=""
    ;;
"rke2")
    cluster_args=""
    ;;
*)
    echo "KUBERNETES_DISTRIBUTION_TYPE=${KUBERNETES_DISTRIBUTION_TYPE} is unknown"
    exit 1
esac