#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

tmp_alerts_yaml=$(mktemp)
trap 'cleanup' EXIT
cleanup() {
    set +e
    rm "${tmp_alerts_yaml}"
}

checkData() {
  if [[ -z "${RANCHER_TOKEN}" ]]; then
      curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-alertmanager:9093/proxy/api/v2/alerts | yq -P - > "${tmp_alerts_yaml}"
  else
      curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-alertmanager:9093/proxy/api/v2/alerts -k -H "Authorization: Bearer ${RANCHER_TOKEN}" | yq -P - > "${tmp_alerts_yaml}"
  fi
}

WAIT_TIMEOUT="${KUBECTL_WAIT_TIMEOUT%s}"
START_TIME=$(date +%s)
while true; do
  checkData
  CHECKS_PASSED=0

  # Check if timeout has been reached
  CURRENT_TIME=$(date +%s)
  ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
  if [[ $ELAPSED_TIME -ge $WAIT_TIMEOUT ]]; then
      echo "ERROR: Timeout reached, condition not met."
      exit 1
  fi

  if [[ $(yq '. | length' "${tmp_alerts_yaml}") != "1" ]]; then
      echo "ERROR: Found the wrong number of alerts in Project Alertmanager, expected only 'Watchdog'"
      cat "${tmp_alerts_yaml}"

    echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
    sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
    continue
  fi
  CHECKS_PASSED=$((CHECKS_PASSED+1))


  if [[ $(yq '.[0].labels.alertname' "${tmp_alerts_yaml}") != "Watchdog" ]]; then
      echo "ERROR: Expected the only alert to be triggered on the Project Alertmanager to be 'Watchdog'"
      cat "${tmp_alerts_yaml}"

      echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
      sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
      continue
  fi
  CHECKS_PASSED=$((CHECKS_PASSED+1))

  if [[ $CHECKS_PASSED -eq 2 ]];then
    # Get final elapsed time
    ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
    break
  fi
done

cat "${tmp_alerts_yaml}"

echo "PASS: Project Alertmanager is up and running"
