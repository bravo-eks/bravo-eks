{{/*
Chart name
*/}}
{{- define "bravo-app.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Full resource name
*/}}
{{- define "bravo-app.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name (include "bravo-app.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "bravo-app.labels" -}}
app.kubernetes.io/name: {{ include "bravo-app.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "bravo-app.selectorLabels" -}}
app.kubernetes.io/name: {{ include "bravo-app.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}