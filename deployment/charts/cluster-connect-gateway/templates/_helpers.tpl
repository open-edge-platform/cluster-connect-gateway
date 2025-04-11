{{- /*
SPDX-FileCopyrightText: (C) 2025 Intel Corporation

SPDX-License-Identifier: Apache-2.0
*/ -}}

{{/*
Expand the name of the chart.
*/}}
{{- define "cluster-connect-gateway.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cluster-connect-gateway.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cluster-connect-gateway.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "cluster-connect-gateway.labels" -}}
helm.sh/chart: {{ include "cluster-connect-gateway.chart" . }}
{{ include "cluster-connect-gateway.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Controller's Metrics service labels
*/}}
{{- define "cluster-connect-gateway.controllerMetricsServiceLabels" -}}
{{- with .Values.controller.metrics.serviceLabels }}
{{- toYaml . }}
{{- end }}
{{- end }}

{{/*
Gateway's service labels
*/}}
{{- define "cluster-connect-gateway.gatewayMetricsServiceLabels" -}}
{{- with .Values.gateway.service.labels }}
{{- toYaml . }}
{{- end }}
{{- end }}


{{/*
Selector labels
*/}}
{{- define "cluster-connect-gateway.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cluster-connect-gateway.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "cluster-connect-gateway.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "cluster-connect-gateway.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
