#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

tmp_dashboards_yaml=$(mktemp)
trap 'cleanup' EXIT
cleanup() {
    set +e
    rm ${tmp_dashboards_yaml}
}

checkData() {
  if [[ -z "${RANCHER_TOKEN}" ]]; then
      curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/search | yq -P - > ${tmp_dashboards_yaml}
  else
      curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/search -k -H "Authorization: Bearer ${RANCHER_TOKEN}" | yq -P - > ${tmp_dashboards_yaml}
  fi
}

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

WAIT_TIMEOUT="${KUBECTL_WAIT_TIMEOUT%s}"
START_TIME=$(date +%s)
while true; do
  checkData

  # Check if timeout has been reached
  CURRENT_TIME=$(date +%s)
  ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
  if [[ $ELAPSED_TIME -ge $WAIT_TIMEOUT ]]; then
      echo "Error: Timeout reached, condition not met."
      exit 1
  fi

  if [[ $(yq '.[].uri' ${tmp_dashboards_yaml} | wc -l | xargs) != "${#expected_dashboards[@]}" ]]; then
    echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
    sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
    continue
  fi

  FOUND_DASHBOARDS=0
  for dashboard in "${expected_dashboards[@]}"; do
      if ! yq '.[].uri' ${tmp_dashboards_yaml} | grep "${dashboard}" 1>/dev/null; then
          echo "ERROR: Expected '${dashboard}' to exist amongst ${#expected_dashboards[@]} dashboards in Project Grafana"
          cat ${tmp_dashboards_yaml}
          echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
          sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
          break
      fi
      FOUND_DASHBOARDS=$((FOUND_DASHBOARDS+1))
  done

  if [[ FOUND_DASHBOARDS -eq 19 ]];then
    # Get final elapsed time
    ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
    break
  fi
done

cat ${tmp_dashboards_yaml}

echo "PASS: Project Grafana has default dashboards loaded"
