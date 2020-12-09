{{- define "name" -}}
{{- default .Release.Name .Values.dex.nameOverride -}}
{{- end }}
