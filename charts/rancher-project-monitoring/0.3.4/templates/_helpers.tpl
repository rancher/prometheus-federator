# Rancher
{{- define "system_default_registry" -}}
{{- if .Values.global.cattle.systemDefaultRegistry -}}
{{- printf "%s/" .Values.global.cattle.systemDefaultRegistry -}}
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

{{/* Prometheus custom resource instance name */}}
{{- define "project-prometheus-stack.prometheus.crname" -}}
{{- if .Values.cleanPrometheusOperatorObjectNames }}
{{- include "project-prometheus-stack.fullname" . }}
{{- else }}
{{- print (include "project-prometheus-stack.fullname" .) "-prometheus" }}
{{- end }}
{{- end }}

{{/* Alertmanager custom resource instance name */}}
{{- define "project-prometheus-stack.alertmanager.crname" -}}
{{- if .Values.cleanPrometheusOperatorObjectNames }}
{{- include "project-prometheus-stack.fullname" . }}
{{- else }}
{{- print (include "project-prometheus-stack.fullname" .) "-alertmanager" -}}
{{- end }}
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
  {{- $values := index . 0 }}
  {{- $kubeVersion := index . 1 }}
  {{- $old := index . 2 }}
  {{- $new := index . 3 }}
  {{- $default := index . 4 }}
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