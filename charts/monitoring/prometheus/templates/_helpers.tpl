{{- define "name" -}}
{{- default .Release.Name .Values.prometheus.nameOverride -}}
{{- end }}
