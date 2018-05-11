package cloudconfig

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMap returns a ConfigMap containing the cloud-config for the supplied data
func ConfigMap(data *resources.TemplateData, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	var cm *corev1.ConfigMap
	if existing != nil {
		cm = existing
	} else {
		cm = &corev1.ConfigMap{}
	}

	configBuffer := bytes.Buffer{}
	configTpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prometheus config template: %v", err)
	}
	if err := configTpl.Execute(&configBuffer, data); err != nil {
		return nil, fmt.Errorf("failed to render prometheus config template: %v", err)
	}

	cm.Name = resources.CloudConfigConfigMapName
	cm.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	cm.Data = map[string]string{
		"config": configBuffer.String(),
	}

	return cm, nil
}

const (
	config = `
{{- if .Cluster.Spec.Cloud.AWS }}
[global]
zone={{ .Cluster.Spec.Cloud.AWS.AvailabilityZone }}
VPC={{ .Cluster.Spec.Cloud.AWS.VPCID }}
kubernetesclustertag={{ .Cluster.Name }}
disablesecuritygroupingress=false
SubnetID={{ .Cluster.Spec.Cloud.AWS.SubnetID }}
RouteTableID={{ .Cluster.Spec.Cloud.AWS.RouteTableID }}
disablestrictzonecheck=true
{{- end }}
{{- if .Cluster.Spec.Cloud.Openstack }}
[Global]
auth-url = "{{ .DC.Spec.Openstack.AuthURL }}"
username = "{{ .Cluster.Spec.Cloud.Openstack.Username }}"
password = "{{ .Cluster.Spec.Cloud.Openstack.Password }}"
domain-name= "{{ .Cluster.Spec.Cloud.Openstack.Domain }}"
tenant-name = "{{ .Cluster.Spec.Cloud.Openstack.Tenant }}"
region = "{{ .DC.Spec.Openstack.Region }}"

[BlockStorage]
trust-device-path = false
bs-version = "v2"
{{- if eq (substr 0 4 (index .Version.Values "k8s-version")) "v1.9" }}
ignore-volume-az = {{ .DC.Spec.Openstack.IgnoreVolumeAZ }}
{{- end }}
{{- if eq (substr 0 5 (index .Version.Values "k8s-version")) "v1.10" }}
ignore-volume-az = {{ .DC.Spec.Openstack.IgnoreVolumeAZ }}
{{- end }}
{{- end }}
`
)
