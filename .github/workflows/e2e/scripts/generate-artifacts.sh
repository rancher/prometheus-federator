#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

case "${KUBERNETES_DISTRIBUTION_TYPE}" in
"k3s")
    components=(
        "k3s-server"
    )
    ;;
"rke")
    components=(
        "kube-controller-manager"
        "kube-scheduler"
        "kube-proxy"
        "kube-etcd"
    )
    ;;
"rke2")
    components=(
        "kube-controller-manager"
        "kube-scheduler"
        "kube-proxy"
        "kube-etcd"
    )
    ;;
*)
    echo "KUBERNETES_DISTRIBUTION_TYPE=${KUBERNETES_DISTRIBUTION_TYPE} is unknown"
    exit 1
esac

ARTIFACT_DIRECTORY=artifacts
MANIFEST_DIRECTORY=${ARTIFACT_DIRECTORY}/manifests
LOG_DIRECTORY=${ARTIFACT_DIRECTORY}/logs

# Manifests
mkdir -p ${MANIFEST_DIRECTORY}
mkdir -p ${MANIFEST_DIRECTORY}/helmcharts
mkdir -p ${MANIFEST_DIRECTORY}/helmreleases
mkdir -p ${MANIFEST_DIRECTORY}/daemonsets
mkdir -p ${MANIFEST_DIRECTORY}/deployments
mkdir -p ${MANIFEST_DIRECTORY}/jobs
mkdir -p ${MANIFEST_DIRECTORY}/statefulsets
mkdir -p ${MANIFEST_DIRECTORY}/pods
mkdir -p ${MANIFEST_DIRECTORY}/projecthelmcharts

kubectl get namespaces -o yaml > ${MANIFEST_DIRECTORY}/namespaces.yaml || true
kubectl get helmcharts -A > ${MANIFEST_DIRECTORY}/helmcharts-list.txt || true
kubectl get services -A > ${MANIFEST_DIRECTORY}/services-list.txt || true

## cattle-monitoring-system ns manifests
kubectl get helmcharts -n cattle-monitoring-system -o yaml > ${MANIFEST_DIRECTORY}/helmcharts/cattle-monitoring-system.yaml || true
kubectl get helmreleases -n cattle-monitoring-system -o yaml > ${MANIFEST_DIRECTORY}/helmreleases/cattle-monitoring-system.yaml || true
kubectl get daemonset -n cattle-monitoring-system -o yaml > ${MANIFEST_DIRECTORY}/daemonsets/cattle-monitoring-system.yaml || true
kubectl get deployment -n cattle-monitoring-system -o yaml > ${MANIFEST_DIRECTORY}/deployments/cattle-monitoring-system.yaml || true
kubectl get job -n cattle-monitoring-system -o yaml > ${MANIFEST_DIRECTORY}/jobs/cattle-monitoring-system.yaml || true
kubectl get statefulset -n cattle-monitoring-system -o yaml > ${MANIFEST_DIRECTORY}/statefulsets/cattle-monitoring-system.yaml || true
kubectl get pods -n cattle-monitoring-system -o yaml > ${MANIFEST_DIRECTORY}/pods/cattle-monitoring-system.yaml || true

## cattle-project-p-example ns manifests
kubectl get deployment -n cattle-project-p-example -o yaml > ${MANIFEST_DIRECTORY}/deployments/cattle-project-p-example.yaml || true
kubectl get projecthelmchart -n cattle-project-p-example -o yaml > ${MANIFEST_DIRECTORY}/projecthelmcharts/cattle-project-p-example.yaml || true
kubectl get statefulset -n cattle-project-p-example -o yaml > ${MANIFEST_DIRECTORY}/statefulsets/cattle-project-p-example.yaml || true
kubectl get pods -n cattle-project-p-example -o yaml > ${MANIFEST_DIRECTORY}/pods/cattle-project-p-example.yaml || true

## cattle-project-p-example-monitoring ns manifests
kubectl get deployment -n cattle-project-p-example-monitoring -o yaml > ${MANIFEST_DIRECTORY}/deployments/cattle-project-p-example-monitoring.yaml || true
kubectl get statefulset -n cattle-project-p-example-monitoring -o yaml > ${MANIFEST_DIRECTORY}/statefulsets/cattle-project-p-example-monitoring.yaml || true
kubectl get pods -n cattle-project-p-example-monitoring -o yaml > ${MANIFEST_DIRECTORY}/pods/cattle-project-p-example-monitoring.yaml || true

# Logs

## Rancher logs
mkdir -p ${LOG_DIRECTORY}/rancher

kubectl logs deployment/rancher-webhook -n cattle-system > ${LOG_DIRECTORY}/rancher/rancher_webhook.log || true
kubectl logs deployment/cattle-cluster-agent -n cattle-system > ${LOG_DIRECTORY}/rancher/cluster_agent.log || true
kubectl logs deployment/system-upgrade-controller -n cattle-system > ${LOG_DIRECTORY}/rancher/upgrade_controller.log || true

mkdir -p ${LOG_DIRECTORY}/rancher-monitoring

## Rancher Monitoring
kubectl logs deployment/rancher-monitoring-operator -n cattle-monitoring-system > ${LOG_DIRECTORY}/rancher-monitoring/prometheus_operator.log || true
kubectl logs statefulset/alertmanager-rancher-monitoring-alertmanager -n cattle-monitoring-system > ${LOG_DIRECTORY}/rancher-monitoring/alertmanager.log || true
kubectl logs statefulset/prometheus-rancher-monitoring-prometheus -n cattle-monitoring-system > ${LOG_DIRECTORY}/rancher-monitoring/prometheus.log || true
kubectl logs deployment/rancher-monitoring-grafana -n cattle-monitoring-system -c grafana > ${LOG_DIRECTORY}/rancher-monitoring/grafana.log || true
kubectl logs deployment/rancher-monitoring-grafana -n cattle-monitoring-system -c grafana-proxy > ${LOG_DIRECTORY}/rancher-monitoring/grafana_proxy.log || true
kubectl logs deployment/rancher-monitoring-grafana -n cattle-monitoring-system -c grafana-sc-dashboard > ${LOG_DIRECTORY}/rancher-monitoring/grafana_sc_dashboard.log || true
kubectl logs deployment/rancher-monitoring-grafana -n cattle-monitoring-system -c grafana-sc-datasources > ${LOG_DIRECTORY}/rancher-monitoring/grafana_sc_datasources.log || true
kubectl logs deployment/rancher-monitoring-grafana -n cattle-monitoring-system -c grafana-init-sc-datasources > ${LOG_DIRECTORY}/rancher-monitoring/grafana_init_sc_datasources.log || true
kubectl logs deployment/rancher-monitoring-kube-state-metrics -n cattle-monitoring-system > ${LOG_DIRECTORY}/rancher-monitoring/kube_state_metrics.log || true
kubectl logs daemonset/rancher-monitoring-prometheus-node-exporter -n cattle-monitoring-system > ${LOG_DIRECTORY}/rancher-monitoring/node_exporter.log || true
kubectl logs deployment/rancher-monitoring-prometheus-adapter -n cattle-monitoring-system > ${LOG_DIRECTORY}/rancher-monitoring/prometheus_adapter.log || true
for component in "${components[@]}"; do
    kubectl logs "daemonset/pushprox-${component}-client" -n cattle-monitoring-system > ${LOG_DIRECTORY}/rancher-monitoring/pushprox-${component}-client.log || true
    kubectl logs "deployment/pushprox-${component}-proxy" -n cattle-monitoring-system > ${LOG_DIRECTORY}/rancher-monitoring/pushprox-${component}-proxy.log || true
done

## Prometheus Federator
mkdir -p ${LOG_DIRECTORY}/prometheus-federator
kubectl logs deployment/prometheus-federator -n cattle-monitoring-system > ${LOG_DIRECTORY}/prometheus-federator/prometheus-federator.log || true
kubectl logs job/helm-install-cattle-project-p-example-monitoring -n cattle-monitoring-system > ${LOG_DIRECTORY}/prometheus-federator/helm_install_project_monitoring_stack.log || true

## Project Monitoring
mkdir -p ${LOG_DIRECTORY}/project-monitoring
kubectl logs statefulset/alertmanager-cattle-project-p-example-m-alertmanager -n cattle-project-p-example > ${LOG_DIRECTORY}/project-monitoring/alertmanager.log || true
kubectl logs statefulset/prometheus-cattle-project-p-example-m-prometheus -n cattle-project-p-example > ${LOG_DIRECTORY}/project-monitoring/prometheus.log || true
kubectl logs deployment/cattle-project-p-example-monitoring-grafana -n cattle-project-p-example -c grafana > ${LOG_DIRECTORY}/project-monitoring/grafana.log || true
kubectl logs deployment/cattle-project-p-example-monitoring-grafana -n cattle-project-p-example -c grafana-proxy > ${LOG_DIRECTORY}/project-monitoring/grafana_proxy.log || true
kubectl logs deployment/cattle-project-p-example-monitoring-grafana -n cattle-project-p-example -c grafana-sc-dashboard > ${LOG_DIRECTORY}/project-monitoring/grafana_sc_dashboard.log || true
kubectl logs deployment/cattle-project-p-example-monitoring-grafana -n cattle-project-p-example -c grafana-sc-datasources > ${LOG_DIRECTORY}/project-monitoring/grafana_sc_datasources.log || true
kubectl logs deployment/cattle-project-p-example-monitoring-grafana -n cattle-project-p-example -c grafana-init-sc-datasources > ${LOG_DIRECTORY}/project-monitoring/grafana_init_sc_datasources.log || true
