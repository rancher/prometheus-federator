#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

tmp_alerts_yaml=$(mktemp)
trap 'cleanup' EXIT
cleanup() {
    set +e
    rm ${tmp_alerts_yaml}
}

if [[ -z "${RANCHER_TOKEN}" ]]; then
    curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-alertmanager:9093/proxy/api/v2/alerts | yq -P - > ${tmp_alerts_yaml}
else
    curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-alertmanager:9093/proxy/api/v2/alerts -k -H "Authorization: Bearer ${RANCHER_TOKEN}" | yq -P - > ${tmp_alerts_yaml}
fi

if [[ $(yq '. | length' "${tmp_alerts_yaml}") != "1" ]]; then
    echo "ERROR: Found the wrong number of alerts in Project Alertmanager, expected only 'Watchdog'"
    cat ${tmp_alerts_yaml}
    exit 1
fi

if [[ $(yq '.[0].labels.alertname' "${tmp_alerts_yaml}") != "Watchdog" ]]; then
    echo "ERROR: Expected the only alert to be triggered on the Project Alertmanager to be 'Watchdog'"
    cat ${tmp_alerts_yaml}
    exit 1
fi

cat ${tmp_alerts_yaml}

echo "PASS: Project Alertmanager is up and running"
