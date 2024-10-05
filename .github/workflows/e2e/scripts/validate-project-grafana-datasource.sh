#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

tmp_datasources_yaml=$(mktemp)
trap 'cleanup' EXIT
cleanup() {
    set +e
    rm ${tmp_datasources_yaml}
}

if [[ -z "${RANCHER_TOKEN}" ]]; then
    curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example-monitoring/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/datasources | yq -P - > ${tmp_datasources_yaml}
else
    curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example-monitoring/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/datasources -k -H "Authorization: Bearer ${RANCHER_TOKEN}" | yq -P - > ${tmp_datasources_yaml}
fi

if [[ $(yq '. | length' ${tmp_datasources_yaml}) != "1" ]]; then
    echo "ERROR: Found the wrong number of datasources in Project Grafana, expected only 'Prometheus'"
    cat ${tmp_datasources_yaml}
    exit 1
fi

if [[ $(yq '.[0].url' ${tmp_datasources_yaml}) != "http://cattle-project-p-example-m-prometheus.cattle-project-p-example-monitoring:9090/" ]]; then
    echo "ERROR: Expected the only datasource to be configured to point to Project Prometheus at Kubernetes DNS http://cattle-project-p-example-m-prometheus.cattle-project-p-example-monitoring:9090/"
    cat ${tmp_datasources_yaml}
    exit 1
fi

if [[ $(yq '.[0].type' ${tmp_datasources_yaml}) != "prometheus" ]]; then
    echo "ERROR: Expected the only datasource to be configured to be of type 'prometheus'"
    cat ${tmp_datasources_yaml}
    exit 1
fi

cat ${tmp_datasources_yaml}

echo "PASS: Project Grafana has the default Prometheus datasource set up to point at Project Prometheus"
