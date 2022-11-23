#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

if ! kubectl -n cattle-project-p-example rollout status statefulset alertmanager-cattle-project-p-example-m-alertmanager --timeout="${KUBECTL_WAIT_TIMEOUT}"; then
    echo "ERROR: Project Alertmanager did not roll out";
    kubectl get pods -n cattle-project-p-example -o yaml;
    kubectl logs statefulset/alertmanager-cattle-project-p-example-m-alertmanager -n cattle-project-p-example;
    exit 1;
fi;

if ! kubectl -n cattle-project-p-example rollout status statefulset prometheus-cattle-project-p-example-m-prometheus --timeout="${KUBECTL_WAIT_TIMEOUT}"; then
    echo "ERROR: Project Prometheus did not roll out";
    kubectl get pods -n cattle-project-p-example -o yaml;
    kubectl logs statefulset/prometheus-cattle-project-p-example-m-prometheus -n cattle-project-p-example;
    exit 1;
fi;

if ! kubectl -n cattle-project-p-example rollout status deployment cattle-project-p-example-monitoring-grafana --timeout="${KUBECTL_WAIT_TIMEOUT}"; then
    echo "ERROR: Project Grafana did not roll out";
    kubectl get pods -n cattle-project-p-example -o yaml;
    echo "GRAFANA";
    kubectl logs deployment/cattle-project-p-example-monitoring-grafana -n cattle-project-p-example -c grafana;
    echo "GRAFANA-PROXY:";
    kubectl logs deployment/cattle-project-p-example-monitoring-grafana -n cattle-project-p-example -c grafana-proxy;
    echo "GRAFANA-SC-DASHBOARD:";
    kubectl logs deployment/cattle-project-p-example-monitoring-grafana -n cattle-project-p-example -c grafana-sc-dashboard;
    echo "GRAFANA-SC-DATASOURCES:";
    kubectl logs deployment/cattle-project-p-example-monitoring-grafana -n cattle-project-p-example -c grafana-sc-datasources;
    exit 1;
fi;

echo "PASS: Project Monitoring Stack is up and running"