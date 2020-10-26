{{- define "dashboard-name" -}}
{{- default (printf "%s-%s" .Release.Name "dashboard") .Values.dashboard.deployment.dashboard.nameOverride -}}
{{- end }}

{{- define "oauth-name" -}}
{{- default (printf "%s-%s" .Release.Name "proxy") .Values.dashboard.deployment.proxy.nameOverride -}}
{{- end }}

{{- define "scraper-name" -}}
{{- default (printf "%s-%s" .Release.Name "scraper") .Values.dashboard.deployment.scraper.nameOverride -}}
{{- end }}
