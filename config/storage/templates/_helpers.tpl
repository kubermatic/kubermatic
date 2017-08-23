{{- define "aws" }}
provisioner: kubernetes.io/aws-ebs
parameters:
  type: {{.Values.StorageType}}
{{- end }}

{{- define "gke" }}
provisioner: kubernetes.io/gce-pd
parameters:
  type: {{.Values.StorageType}}
  zone: {{.Values.StorageZone}}
{{- end }}

{{- define "openstack-cinder" }}
provisioner: kubernetes.io/cinder
parameters:
  type: {{.Values.StorageType}}
  availability: {{.Values.StorageZone}}
{{- end }}

{{- define "bare-metal" }}
provisioner: kubernetes.io/glusterfs
parameters:
  resturl: {{ .Values.StorageURL | quote }}
  restauthenabled: "false"
{{- end }}
