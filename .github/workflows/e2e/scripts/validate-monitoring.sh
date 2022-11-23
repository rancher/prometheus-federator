#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

if ! kubectl -n cattle-monitoring-system rollout status deployment rancher-monitoring-operator --timeout="${KUBECTL_WAIT_TIMEOUT}"; then 
    echo "ERROR: Prometheus Operator did not roll out";
    kubectl get pods -n cattle-monitoring-system -o yaml;
    kubectl logs deployment/rancher-monitoring-operator -n cattle-monitoring-system;
    exit 1;
fi;

if ! kubectl -n cattle-monitoring-system rollout status statefulset alertmanager-rancher-monitoring-alertmanager --timeout="${KUBECTL_WAIT_TIMEOUT}"; then 
    echo "ERROR: Cluster Alertmanager did not roll out";
    kubectl get pods -n cattle-monitoring-system -o yaml;
    kubectl logs statefulset/alertmanager-rancher-monitoring-alertmanager -n cattle-monitoring-system;
    exit 1; 
fi;

if ! kubectl -n cattle-monitoring-system rollout status statefulset prometheus-rancher-monitoring-prometheus --timeout="${KUBECTL_WAIT_TIMEOUT}"; then 
    echo "ERROR: Cluster Prometheus did not roll out";
    kubectl get pods -n cattle-monitoring-system -o yaml;
    kubectl logs statefulset/prometheus-rancher-monitoring-prometheus -n cattle-monitoring-system;
    exit 1;
fi;

if ! kubectl -n cattle-monitoring-system rollout status deployment rancher-monitoring-grafana --timeout="${KUBECTL_WAIT_TIMEOUT}"; then 
    echo "ERROR: Cluster Grafana did not roll out";
    kubectl get pods -n cattle-monitoring-system -o yaml;
    echo "GRAFANA";
    kubectl logs deployment/rancher-monitoring-grafana -n cattle-monitoring-system -c grafana;
    echo "GRAFANA-PROXY:";
    kubectl logs deployment/rancher-monitoring-grafana -n cattle-monitoring-system -c grafana-proxy;
    echo "GRAFANA-SC-DASHBOARD:";
    kubectl logs deployment/rancher-monitoring-grafana -n cattle-monitoring-system -c grafana-sc-dashboard;
    echo "GRAFANA-SC-DATASOURCES:";
    kubectl logs deployment/rancher-monitoring-grafana -n cattle-monitoring-system -c grafana-sc-datasources;
    exit 1;
fi;

if ! kubectl -n cattle-monitoring-system rollout status deployment rancher-monitoring-kube-state-metrics --timeout="${KUBECTL_WAIT_TIMEOUT}"; then 
    echo "ERROR: Kube State Metrics did not roll out";
    kubectl get pods -n cattle-monitoring-system -o yaml;
    kubectl logs deployment/rancher-monitoring-kube-state-metric -n cattle-monitoring-system;
    exit 1;
fi;

if ! kubectl -n cattle-monitoring-system rollout status daemonset rancher-monitoring-prometheus-node-exporter --timeout="${KUBECTL_WAIT_TIMEOUT}"; then 
    echo "ERROR: Node Exporter did not roll out";
    kubectl get pods -n cattle-monitoring-system -o yaml;
    kubectl logs daemonset/rancher-monitoring-prometheus-node-exporter -n cattle-monitoring-system;
    exit 1;
fi;

if ! kubectl -n cattle-monitoring-system rollout status deployment rancher-monitoring-prometheus-adapter --timeout="${KUBECTL_WAIT_TIMEOUT}"; then 
    echo "ERROR: Prometheus Adapter did not roll out";
    kubectl get pods -n cattle-monitoring-system -o yaml;
    kubectl logs deployment/rancher-monitoring-prometheus-adapter -n cattle-monitoring-system;
    exit 1;
fi;

if ! kubectl -n cattle-monitoring-system rollout status daemonset pushprox-k3s-server-client --timeout="${KUBECTL_WAIT_TIMEOUT}"; then 
    echo "ERROR: Pushprox Client did not roll out";
    kubectl get pods -n cattle-monitoring-system -o yaml;
    kubectl logs daemonset/pushprox-k3s-server-client -n cattle-monitoring-system;
    exit 1;
fi;

if ! kubectl -n cattle-monitoring-system rollout status deployment pushprox-k3s-server-proxy --timeout="${KUBECTL_WAIT_TIMEOUT}"; then 
    echo "ERROR: Pushprox Proxy did not roll out";
    kubectl get pods -n cattle-monitoring-system -o yaml;
    kubectl logs deployment/pushprox-k3s-server-proxy -n cattle-monitoring-system;
    exit 1;
fi;

echo "PASS: Rancher Monitoring is up and running"