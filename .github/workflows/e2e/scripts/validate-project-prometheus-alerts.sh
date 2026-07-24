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

# Define allowed alerts
# TODO: consider if this should also test based on context of what container? "Name:container" maybe?
ALLOWED_ALERTS=("Watchdog" "InfoInhibitor" "PrometheusOutOfOrderTimestamps")

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

  # Extract alert names from the YAML
  ALERT_NAMES=($(yq '.[].labels.alertname' "${tmp_alert_rules_yaml}"))

  # Count alerts
  ALERT_COUNT=${#ALERT_NAMES[@]}
  if (( ALERT_COUNT == 0 || ALERT_COUNT > 3 )); then
    echo "ERROR: Found the wrong number of alerts in Project Prometheus, expected only 'Watchdog'"
    echo "ALERT RULES:"
    cat "${tmp_alert_rules_yaml}"

    echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
    sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
    continue
  fi
  CHECKS_PASSED=$((CHECKS_PASSED+1))

  # Ensure "Watchdog" is present
  WATCHDOG_PRESENT=false
  for alert in "${ALERT_NAMES[@]}"; do
      if [[ "$alert" == "Watchdog" ]]; then
          WATCHDOG_PRESENT=true
          break
      fi
  done

  if [[ "$WATCHDOG_PRESENT" == false ]]; then
      echo "ERROR: Expected the at least one alert triggered on the Project Prometheus to be 'Watchdog'"
      echo "ALERT RULES:"
      cat "${tmp_alert_rules_yaml}"

      echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
      sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
      continue
  fi
  CHECKS_PASSED=$((CHECKS_PASSED+1))

  # Check if all alerts are in the allowed list
  for alert in "${ALERT_NAMES[@]}"; do
      FOUND=false
      for allowed in "${ALLOWED_ALERTS[@]}"; do
          if [[ "$alert" == "$allowed" ]]; then
              FOUND=true
              break
          fi
      done
      if [[ "$FOUND" == false ]]; then
          echo "ERROR: Unexpected alert (${alert}) found that is not defined in ALLOWED_ALERTS"
          echo "ALERT RULES:"
          cat "${tmp_alert_rules_yaml}"

          echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
          sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
          continue 2  # Skip to next outer loop iteration
      fi
  done
  CHECKS_PASSED=$((CHECKS_PASSED+1))

  if [[ $CHECKS_PASSED -eq 3 ]];then
    # Get final elapsed time
    ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
    break
  fi
done

cat "${tmp_alert_rules_yaml}";

echo "PASS: Project Prometheus has exactly one alert (Watchdog) active"
