{{/*
Kubernetes version
*/}}
{{- define "chart.KubernetesVersion" -}}
{{- $version := .Capabilities.KubeVersion.Version | regexFind "v[0-9]+\\.[0-9]+\\.[0-9]+" | trimPrefix "v" -}}
{{- printf "%s" $version -}}
{{- end -}}
