#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

API_SERVER_URL=http://localhost:${APISERVER_PORT:-8001}

curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-alertmanager:9093/proxy/api/v2/alerts | yq -P - > alerts.yaml;

if [[ $(yq '. | length' alerts.yaml) != "1" ]]; then
    echo "ERROR: Found the wrong number of alerts in Project Alertmanager, expected only 'Watchdog'";
    cat alerts.yaml;
    exit 1;
fi;

if [[ $(yq '.[0].labels.alertname' alerts.yaml) != "Watchdog" ]]; then
    echo "ERROR: Expected the only alert to be triggered on the Project Alertmanager to be 'Watchdog'";
    cat alerts.yaml;
    exit 1;
fi;

cat alerts.yaml;

echo "PASS: Project Alertmanager is up and running";