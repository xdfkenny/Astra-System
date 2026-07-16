{{- define "astra.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "astra.fullname" -}}
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

{{- define "astra.labels" -}}
helm.sh/chart: "{{ .Chart.Name }}-{{ .Chart.Version }}"
app.kubernetes.io/name: {{ include "astra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "astra.image" -}}
{{ .Values.global.registry }}/{{ .repository }}:{{ .tag | default .Values.global.imageTag }}
{{- end }}

{{- define "astra.selectorLabels" -}}
app.kubernetes.io/name: {{ include "astra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "astra.serviceAccountName" -}}
{{ include "astra.fullname" . }}
{{- end }}

{{- define "astra.natsUrl" -}}
{{- if .Values.global.nats.url }}{{ .Values.global.nats.url }}{{ else }}nats://{{ .Release.Name }}-nats:4222{{ end }}
{{- end }}

{{- define "astra.otelEndpoint" -}}
{{- if .Values.global.otel.endpoint }}{{ .Values.global.otel.endpoint }}{{ else }}http://{{ .Release.Name }}-otel-collector:4317{{ end }}
{{- end }}
