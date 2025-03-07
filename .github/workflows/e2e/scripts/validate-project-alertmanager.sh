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

  ALERT_COUNT=$(yq '. | length' "${tmp_alerts_yaml}")
  if [[ $ALERT_COUNT -gt 3 ]]; then
      echo "ERROR: Found too many alerts in Project Alertmanager. Expected at most 3."
      cat "${tmp_alerts_yaml}"

      echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
      sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
      continue
  fi
  CHECKS_PASSED=$((CHECKS_PASSED+1))

  # Gather alert names into an array
  ALERT_NAMES=($(yq '.[].labels.alertname' "${tmp_alerts_yaml}"))

  # Verify watchdog exists in list of Alert names
  if ! printf '%s\n' "$ALERT_NAMES[@]" | grep -Fxq "Watchdog"; then
      echo "ERROR: Expected the 'Watchdog' alert to be triggered on the Project Alertmanager"
      cat "${tmp_alerts_yaml}"

      echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
      sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
      continue
  fi
  CHECKS_PASSED=$((CHECKS_PASSED+1))

  if [[ $ALERT_COUNT -gt 1 ]]; then
    ALLOWED_ALERTS=("InfoInhibitor" "PrometheusOutOfOrderTimestamps")
    # Remove Watchdog from the list
    filteredArray=()
    for item in "${ALERT_NAMES[@]}"; do
      [[ "$item" != "Watchdog" ]] && filteredArray+=("$item")
    done

    # Now verify that the only items in `filteredArray` are also in ALLOWED_ALERTS
    OKAY=true
    for alert in "${filteredArray[@]}"; do
        found=false
        for allowed in "${ALLOWED_ALERTS[@]}"; do
            if [[ "$alert" == "$allowed" ]]; then
                found=true
                break
            fi
        done

        if [[ "$found" == false ]]; then
            OKAY=false
        fi
    done

    if [[ $OKAY == false ]]; then
      echo "ERROR: Unexpected alert found in active alerts list."
      cat "${tmp_alerts_yaml}"

      echo "Retrying in $DEFAULT_SLEEP_TIMEOUT_SECONDS seconds..."
      sleep "$DEFAULT_SLEEP_TIMEOUT_SECONDS"
      continue
    fi
  fi
  CHECKS_PASSED=$((CHECKS_PASSED+1))

  if [[ $CHECKS_PASSED -eq 3 ]];then
    # Get final elapsed time
    ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
    break
  fi
done

cat "${tmp_alerts_yaml}"

echo "PASS: Project Alertmanager is up and running"
