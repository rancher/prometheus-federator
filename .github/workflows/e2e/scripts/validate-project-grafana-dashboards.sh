#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/search  | yq -P - > dashboards.yaml;
expected_dashboards=(
    db/alertmanager-overview
    db/grafana-overview
    db/kubernetes-compute-resources-namespace-pods
    db/kubernetes-compute-resources-namespace-workloads
    db/kubernetes-compute-resources-node-pods
    db/kubernetes-compute-resources-pod
    db/kubernetes-compute-resources-project
    db/kubernetes-compute-resources-workload
    db/kubernetes-networking-namespace-pods
    db/kubernetes-networking-namespace-workload
    db/kubernetes-networking-pod
    db/kubernetes-networking-project
    db/kubernetes-networking-workload
    db/kubernetes-persistent-volumes
    db/prometheus-overview
    db/rancher-pod
    db/rancher-pod-containers
    db/rancher-workload
    db/rancher-workload-pods
);

if [[ $(yq '.[].uri' dashboards.yaml | wc -l | xargs) != "${#expected_dashboards[@]}" ]]; then
    echo "ERROR: Found the wrong number of dashboards in Project Grafana, expected only the following: ${expected_dashboards[@]}";
    cat dashboards.yaml;
    exit 1;
fi;      

for dashboard in "${expected_dashboards[@]}"; do
    if ! yq '.[].uri' dashboards.yaml | grep "${dashboard}" 1>/dev/null; then
        echo "ERROR: Expected '${dashboard}' to exist amongst ${#expected_dashboards[@]} dashboards in Project Grafana";
        cat dashboards.yaml;
        exit 1;
    fi;
done;

cat dashboards.yaml;

echo "PASS: Project Grafana has default dashboards loaded";
