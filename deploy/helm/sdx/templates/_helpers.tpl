{{/*
Expand the name of the chart.
*/}}
{{- define "sdx.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "sdx.fullname" -}}
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
{{- define "sdx.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "sdx.labels" -}}
helm.sh/chart: {{ include "sdx.chart" . }}
{{ include "sdx.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: neuroforge-sdx
{{- end }}

{{/*
Selector labels
*/}}
{{- define "sdx.selectorLabels" -}}
app.kubernetes.io/name: {{ include "sdx.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Service account name
*/}}
{{- define "sdx.serviceAccountName" -}}
{{- if .Values.gateway.serviceAccount.create }}
{{- default (include "sdx.fullname" .) .Values.gateway.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.gateway.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Image reference
*/}}
{{- define "sdx.gateway.image" -}}
{{- $registry := .Values.global.imageRegistry -}}
{{- $repo := .Values.gateway.image.repository -}}
{{- $tag := .Values.gateway.image.tag | default .Chart.AppVersion -}}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry $repo $tag }}
{{- else }}
{{- printf "%s:%s" $repo $tag }}
{{- end }}
{{- end }}
