{{/* vim: set filetype=mustache: */}}

{{/*
Create path for given chart in the provided git repository
Path would be - base path + kkp version + /charts + specific name of the chart 
*/}}
{{- define "kkp.chart.pathprefix" -}}
{{ if .Values.kkpChartsInCustomRepo }}
{{- printf "%s" .Values.kkpBasePath -}}
{{ else }}
{{- printf "." -}}
{{ end  }}
{{- end -}}

{{- define "git-tag-version" -}}
{{ .Values.environment }}-kkp-{{ .Values.kkpVersion }}
{{- end -}}

{{- define "argo-cd-apps.env-specific-values-file.path" -}}
{{- printf "%s/%s" .Values.environment .Values.envSpecificValuesFileName -}}
{{- end -}}

{{- define "argo-cd-apps.seed-override-values-file.path" -}}
{{- printf "%s/%s/%s" .Values.environment .Values.seed .Values.seedOverrideValuesFileName -}}
{{- end -}}

{{- define "argo-cd-apps.env-specific-kkp-settings.path" -}}
{{- printf "%s/%s" .Values.environment .Values.envSpecificSettingFolderName -}}
{{- end -}}

{{- define "argo-cd-apps.user-mla-values-file.path" -}}
{{- printf "%s/%s/%s" .Values.environment .Values.seed .Values.userMlaValuesFileName -}}
{{- end -}}

