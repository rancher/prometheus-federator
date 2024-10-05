#!/bin/bash
set -e
set -x

source $(dirname $0)/entry

cd $(dirname $0)/../../../..

tmp_dashboards_yaml=$(mktemp)
tmp_queries_yaml=$(mktemp)
trap 'cleanup' EXIT
cleanup() {
    set +e
    rm ${tmp_dashboards_yaml}
    rm ${tmp_queries_yaml}
}

if [[ -z "${RANCHER_TOKEN}" ]]; then
    curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example-monitoring/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/search | yq -P - > ${tmp_dashboards_yaml}
else
    curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example-monitoring/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/search -k -H "Authorization: Bearer ${RANCHER_TOKEN}" | yq -P - > ${tmp_dashboards_yaml}
fi

dashboards=$(yq '.[].uri' ${tmp_dashboards_yaml})

# Collect all queries
for dashboard in ${dashboards[@]}; do
    dashboard_uid=$(yq ".[] | select(.uri==\"${dashboard}\") | .uid" ${tmp_dashboards_yaml});
    if [[ -z "${RANCHER_TOKEN}" ]]; then
        dashboard_json=$(curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example-monitoring/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/dashboards/uid/${dashboard_uid} | yq '.dashboard' -)
    else
        dashboard_json=$(curl -s ${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example-monitoring/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/dashboards/uid/${dashboard_uid} -k -H "Authorization: Bearer ${RANCHER_TOKEN}" | yq '.dashboard' -)
    fi
    # TODO: Fix this to actually recursively utilize Grafana dashboard's yaml structure
    # Today, it just looks for .expr entries in .panels[], .panels[].panels[], and .rows[].panels[], which should cover all dashboards in Monitoring today
    echo "${dashboard_json}" | yq ".panels[].targets[].expr | { \"${dashboard}/\"+(parent|parent|parent|.title|sub(\" \", \"_\"))+\"_query\"+(parent|path|.[-1]) : (. | sub(\"\n\", \"\")) }" - >> ${tmp_queries_yaml}
    echo "${dashboard_json}" | yq ".panels[] | .panels[].targets[].expr | { \"${dashboard}/\"+(parent|parent|parent|parent|parent|.title|sub(\" \", \"_\"))+\"/\"+(parent|parent|parent|.title|sub(\" \", \"_\"))+\"_query\"+(parent|path|.[-1]) : (. | sub(\"\n\", \"\")) }" - >> ${tmp_queries_yaml}
    echo "${dashboard_json}" | yq ".rows[] | .panels[].targets[].expr | { \"${dashboard}/\"+(parent|parent|parent|.title|sub(\" \", \"_\"))+\"_query\"+(parent|path|.[-1]) : (. | sub(\"\n\", \"\")) }" - >> ${tmp_queries_yaml}
done

echo ""

exclude_queries=(
    # Grafana Alerts
    "db/grafana-overview/Firing_Alerts_query0"

    # CPU Throttling Metrics
    "db/kubernetes-compute-resources-pod/CPU_Throttling_query0"
    "db/rancher-pod/CPU_Utilization_query0"
    "db/rancher-pod/CPU_Utilization_query1"
    "db/rancher-pod/CPU_Utilization_query3"
    "db/rancher-pod-containers/CPU_Utilization_query0"
    "db/rancher-pod-containers/CPU_Utilization_query1"
    "db/rancher-pod-containers/CPU_Utilization_query3"
    "db/rancher-workload/CPU_Utilization_query0"
    "db/rancher-workload/CPU_Utilization_query1"
    "db/rancher-workload/CPU_Utilization_query3"
    "db/rancher-workload-pods/CPU_Utilization_query0"
    "db/rancher-workload-pods/CPU_Utilization_query1"
    "db/rancher-workload-pods/CPU_Utilization_query3"


    # Persistent Volume Metrics
    "db/kubernetes-persistent-volumes/Volume_Space_Usage_query0"
    "db/kubernetes-persistent-volumes/Volume_Space_Usage_query1"
    "db/kubernetes-persistent-volumes/Volume_Space_Usage_query0"
    "db/kubernetes-persistent-volumes/Volume_inodes_Usage_query0"
    "db/kubernetes-persistent-volumes/Volume_inodes_Usage_query1"
    "db/kubernetes-persistent-volumes/Volume_inodes_Usage_query0"

    # Flakey Tests
    "db/kubernetes-compute-resources-namespace-pods/IOPS(Reads+Writes)_query0"
    "db/kubernetes-compute-resources-namespace-pods/Current_Storage_IO_query0"
    "db/kubernetes-compute-resources-namespace-pods/Current_Storage_IO_query1"
    "db/kubernetes-compute-resources-namespace-pods/Current_Storage_IO_query2"
    "db/kubernetes-compute-resources-pod/IOPS_query0"
    "db/kubernetes-compute-resources-pod/IOPS_query1"
    "db/kubernetes-compute-resources-pod/IOPS(Reads+Writes)_query0"
    "db/kubernetes-compute-resources-pod/Current_Storage_IO_query0"
    "db/kubernetes-compute-resources-pod/Current_Storage_IO_query1"
    "db/kubernetes-compute-resources-pod/Current_Storage_IO_query2"
    "db/kubernetes-compute-resources-project/IOPS(Reads+Writes)_query0"
    "db/kubernetes-compute-resources-project/Current_Storage_IO_query0"
    "db/kubernetes-compute-resources-project/Current_Storage_IO_query1"
    "db/kubernetes-compute-resources-project/Current_Storage_IO_query2"
    "db/kubernetes-compute-resources-namespace-pods/Memory_Quota_query7"
    "db/kubernetes-compute-resources-node-pods/Memory_Quota_query7"
    "db/kubernetes-compute-resources-pod/Memory_Quota_query7")


unset FAILED
for query_key in $(yq "keys" ${tmp_queries_yaml} | cut -d' ' -f2-); do
    unset skip
    for exclude_query in "${exclude_queries[@]}"; do
        if [[ "${query_key}" == "${exclude_query}" ]]; then
            skip=1
            break
        fi
    done
    [[ -n "${skip}" ]] && echo "WARN: Skipping ${query_key}" && echo "" && continue

    query=$(yq ".[\"${query_key}\"]" ${tmp_queries_yaml})
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
    if [[ -z "${RANCHER_TOKEN}" ]]; then
        query_response=$(curl -s "${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example-monitoring/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/ds/query" -H 'content-type: application/json' --data-raw "${query_body}")
    else
        query_response=$(curl -s "${API_SERVER_URL}/api/v1/namespaces/cattle-project-p-example-monitoring/services/http:cattle-project-p-example-monitoring-grafana:80/proxy/api/ds/query" -H 'content-type: application/json' --data-raw "${query_body}" -k -H "Authorization: Bearer ${RANCHER_TOKEN}")
    fi
    if [[ "$(echo ${query_response} | yq '.message == "bad request data"')" == "true" ]]; then
        # echo "QUERY: ${query}"
        echo "INTERNAL ERROR: Request to /api/ds/query failed due to malformed request: ${query_response}"
        echo "QUERY BODY: ${query_body}"
        echo ""
        FAILED=1
        continue
    fi
    if [[ "$(echo ${query_response} | yq '.results.A.frames | length' -)" == "0" ]]; then
        # echo "QUERY: ${query}"
        echo "ERROR: No data was found for query ${query_key}: ${query_response}"
        echo ""
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

echo "PASS: Project Grafana has default dashboards loaded"
