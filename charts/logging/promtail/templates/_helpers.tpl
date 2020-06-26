{{- define "promtail.name" -}}
{{- default .Release.Name .Values.promtail.nameOverride -}}
{{- end }}

{{- define "promtail.fullname" -}}
{{- default .Release.Name .Values.promtail.nameOverride -}}
{{- end }}

{{- define "promtail.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{- define "promtail.serviceAccountName" -}}
{{- default .Release.Name .Values.promtail.nameOverride -}}
{{- end }}

#{{- define "loki.serviceName" -}}
#{{- $name := default "loki" .Values.promtail.loki.nameOverride -}}
#{{- end }}
