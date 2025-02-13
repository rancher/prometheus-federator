#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

tmp_rules_yaml=$(mktemp)
tmp_alert_rules_yaml=$(mktemp)
trap 'cleanup' EXIT
cleanup() {
    set +e
    rm "${tmp_rules_yaml}"
    rm "${tmp_alert_rules_yaml}"
}

checkData() {
  if [[ -z "${RANCHER_TOKEN}" ]]; then
      curl -s "${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-prometheus:9090/proxy/api/v1/alerts" | yq -P - > "${tmp_rules_yaml}"
  else
      curl -s "${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-m-prometheus:9090/proxy/api/v1/alerts" -k -H "Authorization: Bearer ${RANCHER_TOKEN}" | yq -P - > "${tmp_rules_yaml}"
  fi

  yq '.data.alerts' "${tmp_rules_yaml}" > "${tmp_alert_rules_yaml}"
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
      echo "Error: Timeout reached, condition not met."
      exit 1
  fi

  if [[ $(yq '. | length' "${tmp_alert_rules_yaml}") != "1" ]]; then
    echo "ERROR: Found the wrong number of alerts in Project Prometheus, expected only 'Watchdog'"
    echo "ALERT RULES:"
    cat "${tmp_alert_rules_yaml}"

    echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
    sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
    continue
  fi
  CHECKS_PASSED=$((CHECKS_PASSED+1))


  if [[ $(yq '.[0].labels.alertname' "${tmp_alert_rules_yaml}") != "Watchdog" ]]; then
      echo "ERROR: Expected the only alert to be triggered on the Project Prometheus to be 'Watchdog'"
      echo "ALERT RULES:"
      cat "${tmp_alert_rules_yaml}"

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

cat "${tmp_alert_rules_yaml}";

echo "PASS: Project Prometheus has exactly one alert (Watchdog) active"
