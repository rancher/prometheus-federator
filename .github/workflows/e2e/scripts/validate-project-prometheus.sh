#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

# Ensure Alerting pipeline works as expected

curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-prometheus:9090/proxy/api/v1/alerts | yq -P - > rules.yaml;
yq '.data.alerts' rules.yaml > alert_rules.yaml;

if [[ $(yq '. | length' alert_rules.yaml) != "1" ]]; then
    echo "ERROR: Found the wrong number of alerts in Project Prometheus, expected only 'Watchdog'";
    echo "ALERT RULES:"
    cat alert_rules.yaml;
    exit 1;
fi;

if [[ $(yq '.[0].labels.alertname' alert_rules.yaml) != "Watchdog" ]]; then
    echo "ERROR: Expected the only alert to be triggered on the Project Prometheus to be 'Watchdog'";
    echo "ALERT RULES:"
    cat alert_rules.yaml;
    exit 1;
fi;

cat alert_rules.yaml;

# Ensure that scrape targets are up and healthy

curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-prometheus:9090/proxy/api/v1/targets | yq -P - > targets.yaml;
yq '.data.activeTargets[] | {.labels.job: .health}' targets.yaml > targets_up.yaml;

echo "TARGETS:";
if [[ $(yq '. | length' targets_up.yaml) != "4" ]]; then
    echo "ERROR: Expected exacty 4 targets to be up in Project Prometheus: federate, cattle-project-p-example-m-alertmanager, cattle-project-p-example-m-prometheus, cattle-project-p-example-monitoring-grafana";
    echo "TARGETS:"
    cat targets_up.yaml;
    exit 1;
fi;

for expected_target in federate cattle-project-p-example-m-alertmanager cattle-project-p-example-m-prometheus cattle-project-p-example-monitoring-grafana; do
    if ! grep "${expected_target}" targets_up.yaml; then
        echo "ERROR: Expected '${expected_target}' to exist amongst 4 targets in Project Prometheus";
        echo "TARGETS:"
        cat targets_up.yaml;
        exit 1;
    fi;
done;

echo "PASS: Project Prometheus is up and running";