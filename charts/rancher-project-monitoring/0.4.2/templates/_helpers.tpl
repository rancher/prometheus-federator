# Rancher
{{- define "system_default_registry" -}}
{{- if .Values.global.cattle.systemDefaultRegistry -}}
{{- printf "%s/" .Values.global.cattle.systemDefaultRegistry -}}
{{- end -}}
{{- end -}}

{{- define "monitoring_registry" -}}
  {{- $temp_registry := (include "system_default_registry" .) -}}
  {{- if $temp_registry -}}
    {{- trimSuffix "/" $temp_registry -}}
  {{- else -}}
    {{- .Values.global.imageRegistry -}}
  {{- end -}}
{{- end -}}

{{/*
https://github.com/helm/helm/issues/4535#issuecomment-477778391
Usage: {{ include "call-nested" (list . "SUBCHART_NAME" "TEMPLATE") }}
e.g. {{ include "call-nested" (list . "grafana" "grafana.fullname") }}
*/}}
{{- define "call-nested" }}
{{- $dot := index . 0 }}
{{- $subchart := index . 1 | splitList "." }}
{{- $template := index . 2 }}
{{- $values := $dot.Values }}
{{- range $subchart }}
{{- $values = index $values . }}
{{- end }}
{{- include $template (dict "Chart" (dict "Name" (last $subchart)) "Values" $values "Release" $dot.Release "Capabilities" $dot.Capabilities) }}
{{- end }}

# Special Exporters
{{- define "exporter.kubeEtcd.enabled" -}}
{{- if or .Values.kubeEtcd.enabled .Values.rkeEtcd.enabled .Values.kubeAdmEtcd.enabled .Values.rke2Etcd.enabled -}}
"true"
{{- end -}}
{{- end }}

{{- define "exporter.kubeControllerManager.enabled" -}}
{{- if or .Values.kubeControllerManager.enabled .Values.rkeControllerManager.enabled .Values.k3sServer.enabled .Values.kubeAdmControllerManager.enabled .Values.rke2ControllerManager.enabled -}}
"true"
{{- end -}}
{{- end }}

{{- define "exporter.kubeScheduler.enabled" -}}
{{- if or .Values.kubeScheduler.enabled .Values.rkeScheduler.enabled .Values.k3sServer.enabled .Values.kubeAdmScheduler.enabled .Values.rke2Scheduler.enabled -}}
"true"
{{- end -}}
{{- end }}

{{- define "exporter.kubeProxy.enabled" -}}
{{- if or .Values.kubeProxy.enabled .Values.rkeProxy.enabled .Values.k3sServer.enabled .Values.kubeAdmProxy.enabled .Values.rke2Proxy.enabled -}}
"true"
{{- end -}}
{{- end }}

{{- define "exporter.kubelet.enabled" -}}
{{- if or .Values.kubelet.enabled .Values.hardenedKubelet.enabled .Values.k3sServer.enabled -}}
"true"
{{- end -}}
{{- end }}

{{- define "exporter.kubeControllerManager.jobName" -}}
{{- if .Values.k3sServer.enabled -}}
k3s-server
{{- else -}}
kube-controller-manager
{{- end -}}
{{- end }}

{{- define "exporter.kubeScheduler.jobName" -}}
{{- if .Values.k3sServer.enabled -}}
k3s-server
{{- else -}}
kube-scheduler
{{- end -}}
{{- end }}

{{- define "exporter.kubeProxy.jobName" -}}
{{- if .Values.k3sServer.enabled -}}
k3s-server
{{- else -}}
kube-proxy
{{- end -}}
{{- end }}

{{- define "exporter.kubelet.jobName" -}}
{{- if .Values.k3sServer.enabled -}}
k3s-server
{{- else -}}
kubelet
{{- end -}}
{{- end }}

{{- define "kubelet.serviceMonitor.resourcePath" -}}
{{- $kubeTargetVersion := default .Capabilities.KubeVersion.GitVersion .Values.kubeTargetVersionOverride }}
{{- if not (eq .Values.kubelet.serviceMonitor.resourcePath "/metrics/resource/v1alpha1") -}}
{{ .Values.kubelet.serviceMonitor.resourcePath }}
{{- else if semverCompare ">=1.20.0-0" $kubeTargetVersion -}}
/metrics/resource
{{- else -}}
/metrics/resource/v1alpha1
{{- end -}}
{{- end }}

{{- define "rancher.serviceMonitor.selector" -}}
{{- if .Values.rancherMonitoring.selector }}
{{ .Values.rancherMonitoring.selector | toYaml }}
{{- else }}
{{- $rancherDeployment := (lookup "apps/v1" "Deployment" "cattle-system" "rancher") }}
{{- if $rancherDeployment }}
matchLabels:
  app: rancher
  chart: {{ index $rancherDeployment.metadata.labels "chart" }}
  release: rancher
{{- end }}
{{- end }}
{{- end }}

# Windows Support

{{/*
Windows cluster will add default taint for linux nodes,
add below linux tolerations to workloads could be scheduled to those linux nodes
*/}}

{{- define "linux-node-tolerations" -}}
- key: "cattle.io/os"
  value: "linux"
  effect: "NoSchedule"
  operator: "Equal"
{{- end -}}

{{- define "linux-node-selector" -}}
{{- if semverCompare "<1.14-0" .Capabilities.KubeVersion.GitVersion -}}
beta.kubernetes.io/os: linux
{{- else -}}
kubernetes.io/os: linux
{{- end -}}
{{- end -}}

# Prometheus Operator

{{/* Comma-delimited list of namespaces that need to be watched to configure Project Prometheus Stack components */}}
{{- define "project-prometheus-stack.projectNamespaceList" -}}
{{ append .Values.global.cattle.projectNamespaces .Release.Namespace | uniq | join "," }}
{{- end }}

{{/* vim: set filetype=mustache: */}}
{{/* Expand the name of the chart. This is suffixed with -alertmanager, which means subtract 13 from longest 63 available */}}
{{- define "project-prometheus-stack.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 50 | trimSuffix "-" -}}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
The components in this chart create additional resources that expand the longest created name strings.
The longest name that gets created adds and extra 37 characters, so truncation should be 63-35=26.
*/}}
{{- define "project-prometheus-stack.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 26 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 26 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 26 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/* Fullname suffixed with operator */}}
{{- define "project-prometheus-stack.operator.fullname" -}}
{{- printf "%s-operator" (include "project-prometheus-stack.fullname" .) -}}
{{- end }}

{{/* Prometheus custom resource instance name */}}
{{- define "project-prometheus-stack.prometheus.crname" -}}
{{- if .Values.cleanPrometheusOperatorObjectNames }}
{{- include "project-prometheus-stack.fullname" . }}
{{- else }}
{{- print (include "project-prometheus-stack.fullname" .) "-prometheus" }}
{{- end }}
{{- end }}

{{/* Prometheus apiVersion for networkpolicy */}}
{{- define "project-prometheus-stack.prometheus.networkPolicy.apiVersion" -}}
{{- print "networking.k8s.io/v1" -}}
{{- end }}

{{/* Alertmanager custom resource instance name */}}
{{- define "project-prometheus-stack.alertmanager.crname" -}}
{{- if .Values.cleanPrometheusOperatorObjectNames }}
{{- include "project-prometheus-stack.fullname" . }}
{{- else }}
{{- print (include "project-prometheus-stack.fullname" .) "-alertmanager" -}}
{{- end }}
{{- end }}

{{/* Fullname suffixed with thanos-ruler */}}
{{- define "project-prometheus-stack.thanosRuler.fullname" -}}
{{- printf "%s-thanos-ruler" (include "project-prometheus-stack.fullname" .) -}}
{{- end }}

{{/* Shortened name suffixed with thanos-ruler */}}
{{- define "project-prometheus-stack.thanosRuler.name" -}}
{{- default (printf "%s-thanos-ruler" (include "project-prometheus-stack.name" .)) .Values.thanosRuler.name -}}
{{- end }}


{{/* Create chart name and version as used by the chart label. */}}
{{- define "project-prometheus-stack.chartref" -}}
{{- replace "+" "_" .Chart.Version | printf "%s-%s" .Chart.Name -}}
{{- end }}

{{/* Generate basic labels */}}
{{- define "project-prometheus-stack.labels" }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: "{{ replace "+" "_" .Chart.Version }}"
app.kubernetes.io/part-of: {{ template "project-prometheus-stack.name" . }}
chart: {{ template "project-prometheus-stack.chartref" . }}
release: {{ $.Release.Name | quote }}
heritage: {{ $.Release.Service | quote }}
{{- if .Values.commonLabels}}
{{ toYaml .Values.commonLabels }}
{{- end }}
{{- end }}

{{/* Create the name of project-prometheus-stack service account to use */}}
{{- define "project-prometheus-stack.operator.serviceAccountName" -}}
{{- if .Values.prometheusOperator.serviceAccount.create -}}
    {{ default (include "project-prometheus-stack.operator.fullname" .) .Values.prometheusOperator.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.prometheusOperator.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/* Create the name of prometheus service account to use */}}
{{- define "project-prometheus-stack.prometheus.serviceAccountName" -}}
{{- if .Values.prometheus.serviceAccount.create -}}
    {{ default (print (include "project-prometheus-stack.fullname" .) "-prometheus") .Values.prometheus.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.prometheus.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/* Create the name of alertmanager service account to use */}}
{{- define "project-prometheus-stack.alertmanager.serviceAccountName" -}}
{{- if .Values.alertmanager.serviceAccount.create -}}
    {{ default (print (include "project-prometheus-stack.fullname" .) "-alertmanager") .Values.alertmanager.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.alertmanager.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/* Create the name of thanosRuler service account to use */}}
{{- define "project-prometheus-stack.thanosRuler.serviceAccountName" -}}
{{- if .Values.thanosRuler.serviceAccount.create -}}
    {{ default (include "project-prometheus-stack.thanosRuler.name" .) .Values.thanosRuler.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.thanosRuler.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Allow the release namespace to be overridden for multi-namespace deployments in combined charts
*/}}
{{- define "project-prometheus-stack.namespace" -}}
  {{- if .Values.namespaceOverride -}}
    {{- .Values.namespaceOverride -}}
  {{- else -}}
    {{- .Release.Namespace -}}
  {{- end -}}
{{- end -}}

{{/*
Use the grafana namespace override for multi-namespace deployments in combined charts
*/}}
{{- define "project-prometheus-stack-grafana.namespace" -}}
  {{- if .Values.grafana.namespaceOverride -}}
    {{- .Values.grafana.namespaceOverride -}}
  {{- else -}}
    {{- .Release.Namespace -}}
  {{- end -}}
{{- end -}}

{{/*
Use the kube-state-metrics namespace override for multi-namespace deployments in combined charts
*/}}
{{- define "project-prometheus-stack-kube-state-metrics.namespace" -}}
  {{- if index .Values "kube-state-metrics" "namespaceOverride" -}}
    {{- index .Values "kube-state-metrics" "namespaceOverride" -}}
  {{- else -}}
    {{- .Release.Namespace -}}
  {{- end -}}
{{- end -}}

{{/*
Use the prometheus-node-exporter namespace override for multi-namespace deployments in combined charts
*/}}
{{- define "project-prometheus-stack-prometheus-node-exporter.namespace" -}}
  {{- if index .Values "prometheus-node-exporter" "namespaceOverride" -}}
    {{- index .Values "prometheus-node-exporter" "namespaceOverride" -}}
  {{- else -}}
    {{- .Release.Namespace -}}
  {{- end -}}
{{- end -}}

{{/* Allow KubeVersion to be overridden. */}}
{{- define "project-prometheus-stack.kubeVersion" -}}
  {{- default .Capabilities.KubeVersion.Version .Values.kubeVersionOverride -}}
{{- end -}}

{{/* Get Ingress API Version */}}
{{- define "project-prometheus-stack.ingress.apiVersion" -}}
  {{- if and (.Capabilities.APIVersions.Has "networking.k8s.io/v1") (semverCompare ">= 1.19-0" (include "project-prometheus-stack.kubeVersion" .)) -}}
      {{- print "networking.k8s.io/v1" -}}
  {{- else if .Capabilities.APIVersions.Has "networking.k8s.io/v1beta1" -}}
    {{- print "networking.k8s.io/v1beta1" -}}
  {{- else -}}
    {{- print "extensions/v1beta1" -}}
  {{- end -}}
{{- end -}}

{{/* Check Ingress stability */}}
{{- define "project-prometheus-stack.ingress.isStable" -}}
  {{- eq (include "project-prometheus-stack.ingress.apiVersion" .) "networking.k8s.io/v1" -}}
{{- end -}}

{{/* Check Ingress supports pathType */}}
{{/* pathType was added to networking.k8s.io/v1beta1 in Kubernetes 1.18 */}}
{{- define "project-prometheus-stack.ingress.supportsPathType" -}}
  {{- or (eq (include "project-prometheus-stack.ingress.isStable" .) "true") (and (eq (include "project-prometheus-stack.ingress.apiVersion" .) "networking.k8s.io/v1beta1") (semverCompare ">= 1.18-0" (include "project-prometheus-stack.kubeVersion" .))) -}}
{{- end -}}

{{/* Get Policy API Version */}}
{{- define "project-prometheus-stack.pdb.apiVersion" -}}
  {{- if and (.Capabilities.APIVersions.Has "policy/v1") (semverCompare ">= 1.21-0" (include "project-prometheus-stack.kubeVersion" .)) -}}
      {{- print "policy/v1" -}}
  {{- else -}}
    {{- print "policy/v1beta1" -}}
  {{- end -}}
  {{- end -}}

{{/* Get value based on current Kubernetes version */}}
{{- define "project-prometheus-stack.kubeVersionDefaultValue" -}}
  {{- $values := index . 0 -}}
  {{- $kubeVersion := index . 1 -}}
  {{- $old := index . 2 -}}
  {{- $new := index . 3 -}}
  {{- $default := index . 4 -}}
  {{- if kindIs "invalid" $default -}}
    {{- if semverCompare $kubeVersion (include "project-prometheus-stack.kubeVersion" $values) -}}
      {{- print $new -}}
    {{- else -}}
      {{- print $old -}}
    {{- end -}}
  {{- else -}}
    {{- print $default }}
  {{- end -}}
{{- end -}}

{{/* Get value for kube-controller-manager depending on insecure scraping availability */}}
{{- define "project-prometheus-stack.kubeControllerManager.insecureScrape" -}}
  {{- $values := index . 0 -}}
  {{- $insecure := index . 1 -}}
  {{- $secure := index . 2 -}}
  {{- $userValue := index . 3 -}}
  {{- include "project-prometheus-stack.kubeVersionDefaultValue" (list $values ">= 1.22-0" $insecure $secure $userValue) -}}
{{- end -}}

{{/* Get value for kube-scheduler depending on insecure scraping availability */}}
{{- define "project-prometheus-stack.kubeScheduler.insecureScrape" -}}
  {{- $values := index . 0 -}}
  {{- $insecure := index . 1 -}}
  {{- $secure := index . 2 -}}
  {{- $userValue := index . 3 -}}
  {{- include "project-prometheus-stack.kubeVersionDefaultValue" (list $values ">= 1.23-0" $insecure $secure $userValue) -}}
{{- end -}}

{{/* Sets default scrape limits for servicemonitor */}}
{{- define "servicemonitor.scrapeLimits" -}}
{{- with .sampleLimit }}
sampleLimit: {{ . }}
{{- end }}
{{- with .targetLimit }}
targetLimit: {{ . }}
{{- end }}
{{- with .labelLimit }}
labelLimit: {{ . }}
{{- end }}
{{- with .labelNameLengthLimit }}
labelNameLengthLimit: {{ . }}
{{- end }}
{{- with .labelValueLengthLimit }}
labelValueLengthLimit: {{ . }}
{{- end }}
{{- end -}}

{{/*
To help compatibility with other charts which use global.imagePullSecrets.
Allow either an array of {name: pullSecret} maps (k8s-style), or an array of strings (more common helm-style).
global:
  imagePullSecrets:
  - name: pullSecret1
  - name: pullSecret2

or

global:
  imagePullSecrets:
  - pullSecret1
  - pullSecret2
*/}}
{{- define "project-prometheus-stack.imagePullSecrets" -}}
{{- range .Values.global.imagePullSecrets }}
  {{- if eq (typeOf .) "map[string]interface {}" }}
- {{ toYaml . | trim }}
  {{- else }}
- name: {{ . }}
  {{- end }}
{{- end }}
{{- end -}}


{{/* Define ingress for all hardened namespaces */}}
{{- define "project-prometheus-stack.hardened.networkPolicy.ingress" -}}
{{- $root := index . 0 }}
{{- $ns := index . 1 }}
{{- if $root.Values.global.networkPolicy.ingress -}}
{{ toYaml $root.Values.global.networkPolicy.ingress }}
{{- end }}
{{- if $root.Values.global.networkPolicy.limitIngressToProject }}
- from:
{{- if $root.Values.global.cattle.projectNamespaceSelector }}
  - namespaceSelector: {{- $root.Values.global.cattle.projectNamespaceSelector | toYaml | nindent 6 }}
{{- end }}
  - namespaceSelector:
      matchLabels:
        kubernetes.io/metadata.name: {{ $ns }}
{{- end }}
{{- end -}}

{{- define "project-prometheus-stack.operator.admission-webhook.dnsNames" }}
{{- $fullname := include "project-prometheus-stack.operator.fullname" . }}
{{- $namespace := include "project-prometheus-stack.namespace" . }}
{{- $fullname }}
{{ $fullname }}.{{ $namespace }}.svc
{{- if .Values.prometheusOperator.admissionWebhooks.deployment.enabled }}
{{ $fullname }}-webhook
{{ $fullname }}-webhook.{{ $namespace }}.svc
{{- end }}
{{- end }}
