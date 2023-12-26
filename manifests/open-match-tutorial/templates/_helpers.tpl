{{/*
Expand the name of the chart.
*/}}
{{- define "open-match-tutorial.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "open-match-tutorial.fullname" -}}
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

{{- define "open-match-tutorial.mmfName" -}}
{{ include "open-match-tutorial.fullname" . }}-mmf
{{- end }}
{{- define "open-match-tutorial.gameserverName" -}}
{{ include "open-match-tutorial.fullname" . }}-gameserver
{{- end }}
{{- define "open-match-tutorial.gamefrontName" -}}
{{ include "open-match-tutorial.fullname" . }}-gamefront
{{- end }}
{{- define "open-match-tutorial.directorName" -}}
{{ include "open-match-tutorial.fullname" . }}-director
{{- end }}


{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "open-match-tutorial.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "open-match-tutorial.labels" -}}
helm.sh/chart: {{ include "open-match-tutorial.chart" . }}
{{ include "open-match-tutorial.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "open-match-tutorial.selectorLabels" -}}
app.kubernetes.io/name: {{ include "open-match-tutorial.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "open-match-tutorial.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "open-match-tutorial.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
