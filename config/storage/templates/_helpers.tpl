{{- define "aws" }}
provisioner: kubernetes.io/aws-ebs
parameters:
  type: {{ .type }}
{{- end }}

{{- define "gke" }}
provisioner: kubernetes.io/gce-pd
parameters:
  type: {{ .type }}
  zone: {{ .zone }}
{{- end }}

{{- define "openstack-cinder" }}
provisioner: kubernetes.io/cinder
parameters:
  type: {{ .type }}
  availability: {{ .zone }}
{{- end }}
