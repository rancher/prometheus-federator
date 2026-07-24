#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

tmp_targets_yaml=$(mktemp)
tmp_targets_up_yaml=$(mktemp)
trap 'cleanup' EXIT
cleanup() {
    set +e
    rm "${tmp_targets_yaml}"
    rm "${tmp_targets_up_yaml}"
}

checkData() {
  if [[ -z "${RANCHER_TOKEN}" ]]; then
      curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-prometheus:9090/proxy/api/v1/targets | yq -P - > "${tmp_targets_yaml}"
  else
      curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-prometheus:9090/proxy/api/v1/targets -k -H "Authorization: Bearer ${RANCHER_TOKEN}" | yq -P - > "${tmp_targets_yaml}"
  fi

  yq '.data.activeTargets[] | {.labels.job: .health}' "${tmp_targets_yaml}" > "${tmp_targets_up_yaml}";
}

WAIT_TIMEOUT="${KUBECTL_WAIT_TIMEOUT%s}"
START_TIME=$(date +%s)
while true; do
  checkData

  # Check if timeout has been reached
  CURRENT_TIME=$(date +%s)
  ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
  if [[ $ELAPSED_TIME -ge $WAIT_TIMEOUT ]]; then
      echo "ERROR: Timeout reached, condition not met."
      exit 1
  fi

  if [[ $(yq '. | length' "${tmp_targets_up_yaml}") != "4" ]]; then
    echo "ERROR: Expected exactly 4 targets but found $(yq '. | length' "${tmp_targets_up_yaml}")."
    echo "Expected Targets in Project Prometheus: federate, cattle-project-p-example-m-alertmanager, cattle-project-p-example-m-prometheus, cattle-project-p-example-monitoring-grafana"
    echo "TARGETS:"
    cat "${tmp_targets_up_yaml}"

    echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
    sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
    continue
  fi

  FOUND_TARGETS=0
  for expected_target in federate cattle-project-p-example-m-alertmanager cattle-project-p-example-m-prometheus cattle-project-p-example-monitoring-grafana; do
      if ! grep "${expected_target}" "${tmp_targets_up_yaml}"; then
          echo "ERROR: Expected '${expected_target}' to exist amongst 4 targets in Project Prometheus"

          echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
          sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
          break
      fi
      if ! grep "${expected_target}" "${tmp_targets_up_yaml}" | grep up; then
          echo "ERROR: Expected '${expected_target}' to exist in 'up' state"

          echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
          sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
          break
      fi
      FOUND_TARGETS=$((FOUND_TARGETS+1))
  done

  if [[ $FOUND_TARGETS -eq 4 ]];then
    # Get final elapsed time
    ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
    break
  fi
done

echo "PASS: Project Prometheus has all targets healthy after ${ELAPSED_TIME}s"
