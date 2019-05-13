{{- define "name" -}}
{{- default .Release.Name .Values.alertmanager.nameOverride -}}
{{- end }}
