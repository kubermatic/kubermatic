{{- define "target_name" -}}
{{- if .Values.alertmanager.auth_settings.oidc.enabled -}}
alertmanager-kubermatic
{{- else -}}
alertmanager-kubermatic-oidc
{{- end -}}
{{- end -}}
