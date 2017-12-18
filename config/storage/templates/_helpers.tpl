{{- define "aws" }}
provisioner: kubernetes.io/aws-ebs
parameters:
  type: {{.Type}}
{{- end }}

{{- define "gke" }}
provisioner: kubernetes.io/gce-pd
parameters:
  type: {{.Type}}
  zone: {{.Zone}}
{{- end }}

{{- define "openstack-cinder" }}
provisioner: kubernetes.io/cinder
parameters:
  type: {{.Type}}
  availability: {{.Zone}}
{{- end }}

{{- define "bare-metal" }}
provisioner: kubernetes.io/glusterfs
parameters:
  resturl: {{ .URL | quote }}
  restauthenabled: "false"
{{- end }}
