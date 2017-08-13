{{- define "aws" }}
provisioner: kubernetes.io/aws-ebs
parameters:
  type: gp2
{{- end }}

{{- define "gke" }}
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-ssd
  zone: {{.Values.StorageZone}}
{{- end }}

{{- define "bare-metal" }}
provisioner: kubernetes.io/glusterfs
parameters:
  resturl: {{ .Values.StorageURL | quote }}
  restauthenabled: "false"
{{- end }}
