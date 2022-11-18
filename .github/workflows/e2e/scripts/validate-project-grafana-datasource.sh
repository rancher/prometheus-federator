#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/datasources  | yq -P - > datasources.yaml;

if [[ $(yq '. | length' datasources.yaml) != "1" ]]; then
    echo "ERROR: Found the wrong number of datasources in Project Grafana, expected only 'Prometheus'";
    cat datasources.yaml;
    exit 1;
fi;

if [[ $(yq '.[0].url' datasources.yaml) != "http://cattle-project-p-example-m-prometheus.cattle-project-p-example:9090/" ]]; then
    echo "ERROR: Expected the only datasource to be configured to point to Project Prometheus at Kubernetes DNS http://cattle-project-p-example-m-prometheus.cattle-project-p-example:9090/";
    cat datasources.yaml;
    exit 1;
fi;

if [[ $(yq '.[0].type' datasources.yaml) != "prometheus" ]]; then
    echo "ERROR: Expected the only datasource to be configured to be of type 'prometheus'";
    cat datasources.yaml;
    exit 1;
fi;

cat datasources.yaml;

echo "PASS: Project Grafana has the default Prometheus datasource set up to point at Project Prometheus";