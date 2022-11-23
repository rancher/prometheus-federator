#!/bin/bash
set -e

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/search  | yq -P - > dashboards.yaml;

rm queries.yaml 1>/dev/null 2>/dev/null || true

dashboards=$(yq '.[].uri' dashboards.yaml)

# Collect all queries
for dashboard in ${dashboards[@]}; do
    dashboard_uid=$(yq ".[] | select(.uri==\"${dashboard}\") | .uid" dashboards.yaml);
    dashboard_json=$(curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/dashboards/uid/${dashboard_uid} | yq '.dashboard' -)
    # TODO: Fix this to actually recursively utilize Grafana dashboard's yaml structure
    # Today, it just looks for .expr entries in .panels[], .panels[].panels[], and .rows[].panels[], which should cover all dashboards in Monitoring today
    echo "${dashboard_json}" | yq ".panels[].targets[].expr | { \"${dashboard}/\"+(parent|parent|parent|.title|sub(\" \", \"_\"))+\"_query\"+(parent|path|.[-1]) : (. | sub(\"\n\", \"\")) }" - >> queries.yaml
    echo "${dashboard_json}" | yq ".panels[] | .panels[].targets[].expr | { \"${dashboard}/\"+(parent|parent|parent|parent|parent|.title|sub(\" \", \"_\"))+\"/\"+(parent|parent|parent|.title|sub(\" \", \"_\"))+\"_query\"+(parent|path|.[-1]) : (. | sub(\"\n\", \"\")) }" - >> queries.yaml
    echo "${dashboard_json}" | yq ".rows[] | .panels[].targets[].expr | { \"${dashboard}/\"+(parent|parent|parent|.title|sub(\" \", \"_\"))+\"_query\"+(parent|path|.[-1]) : (. | sub(\"\n\", \"\")) }" - >> queries.yaml
done;

# echo "QUERIES:";
# cat queries.yaml;

echo ""

exclude_queries=(
    # Grafana Alerts
    "db/grafana-overview/Firing_Alerts_query0"

    # CPU Throttling Metrics
    "db/kubernetes-compute-resources-pod/CPU_Throttling_query0"
    "db/rancher-pod/CPU_Utilization_query0"
    "db/rancher-pod-containers/CPU_Utilization_query0"
    "db/rancher-workload/CPU_Utilization_query0"
    "db/rancher-workload-pods/CPU_Utilization_query0"

    # Persistent Volume Metrics
    "db/kubernetes-persistent-volumes/Volume_Space_Usage_query0"
    "db/kubernetes-persistent-volumes/Volume_Space_Usage_query1"
    "db/kubernetes-persistent-volumes/Volume_Space_Usage_query0"
    "db/kubernetes-persistent-volumes/Volume_inodes_Usage_query0"
    "db/kubernetes-persistent-volumes/Volume_inodes_Usage_query1"
    "db/kubernetes-persistent-volumes/Volume_inodes_Usage_query0"
)


unset FAILED
for query_key in $(yq "keys" queries.yaml | cut -d' ' -f2-); do
    unset skip
    for exclude_query in "${exclude_queries[@]}"; do
        if [[ "${query_key}" == "${exclude_query}" ]]; then
            skip=1;
            break;
        fi
    done
    [[ -n "${skip}" ]] && echo "WARN: Skipping ${query_key}" && echo "" && continue

    query=$(yq ".[\"${query_key}\"]" queries.yaml)
    normalized_query="${query}"
    normalized_query=$(echo "${normalized_query}" | sed 's:$interval:5m:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$resolution:1m:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$__rate_interval:5m:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:=\"$namespace\":=~\".*\":g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$namespace:.*:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:=\"$type\":=~\".*\":g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$type:.*:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:=\"$kind\":=~\".*\":g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$kind:.*:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:=\"$instance\":=~\".*\":g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$instance:.*:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:=\"$node\":=~\".*\":g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$node:.*:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:=\"$workload\":=~\".*\":g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$workload:.*:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:=\"$pod\":=~\".*\":g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$pod:.*:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:=\"$job\":=~\".*\":g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$job:.*:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:=\"$service\":=~\".*\":g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$service:.*:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:=\"$volume\":=~\".*\":g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$volume:.*:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:=\"$integration\":=~\".*\":g')
    normalized_query=$(echo "${normalized_query}" | sed 's:$integration:.*:g')
    normalized_query=$(echo "${normalized_query}" | sed 's:":\\":g')
    # normalized_query=$(echo "${normalized_query}" | sed -e 's:$[a-z]*:.*:g')
    query_body="$(cat <<EOF
{
    "queries": [
        {
        "expr": "${normalized_query}",
        "datasource": {
            "uid": "prometheus",
            "type": "prometheus"
        },
        "intervalMs": 60000,
        "maxDataPoints": 1
        }
    ],
    "from": "now-5m",
    "to": "now"
}
EOF
    )"
    query_response=$(curl -s "${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/ds/query" -H 'content-type: application/json' --data-raw "${query_body}")
    if [[ "$(echo ${query_response} | yq '.message == "bad request data"')" == "true" ]]; then
        # echo "QUERY: ${query}"
        echo "INTERNAL ERROR: Request to /api/ds/query failed due to malformed request: ${query_response}"
        echo "QUERY BODY: ${query_body}"
        FAILED=1
        continue
    fi
    if [[ "$(echo ${query_response} | yq '.results.A.frames | length' -)" == "0" ]]; then
        # echo "QUERY: ${query}"
        echo "ERROR: No data was found for query ${query_key}: ${query_response}"
        FAILED=1
        continue
    fi
    echo "PASS: Data found for ${query_key}"
    echo ""
done

if [[ -n ${FAILED} ]]; then
    echo "FAILED: Some queries do not have data collected, see logs above for more details"
    exit 1
fi

echo "PASS: Project Grafana has default dashboards loaded";
