#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

tmp_targets_yaml=$(mktemp)
tmp_targets_up_yaml=$(mktemp)
trap 'cleanup' EXIT
cleanup() {
    set +e
    rm ${tmp_targets_yaml}
    rm ${tmp_targets_up_yaml}
}

if [[ -z "${RANCHER_TOKEN}" ]]; then
    curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-prometheus:9090/proxy/api/v1/targets | yq -P - > ${tmp_targets_yaml}
else
    curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-prometheus:9090/proxy/api/v1/targets -k -H "Authorization: Bearer ${RANCHER_TOKEN}" | yq -P - > ${tmp_targets_yaml}
fi

yq '.data.activeTargets[] | {.labels.job: .health}' ${tmp_targets_yaml} > ${tmp_targets_up_yaml};

echo "TARGETS:";
if [[ $(yq '. | length' ${tmp_targets_up_yaml}) != "4" ]]; then
    echo "ERROR: Expected exactly 4 targets but found $(yq '. | length' ${tmp_targets_up_yaml})."
    echo "Expected Targets in Project Prometheus: federate, cattle-project-p-example-m-alertmanager, cattle-project-p-example-m-prometheus, cattle-project-p-example-monitoring-grafana"
    echo "TARGETS:"
    cat ${tmp_targets_up_yaml}
    exit 1
fi

for expected_target in federate cattle-project-p-example-m-alertmanager cattle-project-p-example-m-prometheus cattle-project-p-example-monitoring-grafana; do
    if ! grep "${expected_target}" ${tmp_targets_up_yaml}; then
        echo "ERROR: Expected '${expected_target}' to exist amongst 4 targets in Project Prometheus"
        echo "TARGETS:"
        cat ${tmp_targets_up_yaml}
        exit 1
    fi
done

echo "PASS: Project Prometheus has all targets healthy"
