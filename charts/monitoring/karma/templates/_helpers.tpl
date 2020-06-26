{{- define "name" -}}
{{- default .Release.Name .Values.karma.nameOverride -}}
{{- end }}
