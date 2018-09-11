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

const (
	name = "cloud-config"
)

// ConfigMap returns a ConfigMap containing the cloud-config for the supplied data
func ConfigMap(data *resources.TemplateData, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	var cm *corev1.ConfigMap
	if existing != nil {
		cm = existing
	} else {
		cm = &corev1.ConfigMap{}
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	cloudConfig, err := CloudConfig(data)
	if err != nil {
		return nil, err
	}

	cm.Name = resources.CloudConfigConfigMapName
	cm.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	cm.Labels = resources.BaseAppLabel(name, nil)
	cm.Data["config"] = cloudConfig
	cm.Data[FakeVMWareUUIDKeyName] = fakeVMWareUUID

	return cm, nil
}

// CloudConfig returns the cloud-config for the supplied data
func CloudConfig(data *resources.TemplateData) (string, error) {
	configBuffer := bytes.Buffer{}
	configTpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(config)
	if err != nil {
		return "", fmt.Errorf("failed to parse cloud config template: %v", err)
	}
	if err := configTpl.Execute(&configBuffer, data); err != nil {
		return "", fmt.Errorf("failed to render cloud config template: %v", err)
	}

	return configBuffer.String(), nil
}

const (
	// FakeVMWareUUIDKeyName is the name of the cloud-config configmap key
	// that holds the fake vmware uuid
	// It is required when activating the vsphere cloud-provider in the controller
	// manager on a non-ESXi host
	// Upstream issue: https://github.com/kubernetes/kubernetes/issues/65145
	FakeVMWareUUIDKeyName = "fakeVmwareUUID"
	fakeVMWareUUID        = "VMware-42 00 00 00 00 00 00 00-00 00 00 00 00 00 00 00"
	config                = `
{{- if .Cluster.Spec.Cloud.AWS }}
[global]
zone={{ .Cluster.Spec.Cloud.AWS.AvailabilityZone | quote }}
VPC={{ .Cluster.Spec.Cloud.AWS.VPCID | quote }}
KubernetesClusterID={{ .Cluster.Name | quote }}
disablesecuritygroupingress=false
SubnetID={{ .Cluster.Spec.Cloud.AWS.SubnetID | quote }}
RouteTableID={{ .Cluster.Spec.Cloud.AWS.RouteTableID | quote }}
disablestrictzonecheck=true
{{- end }}
{{- if .Cluster.Spec.Cloud.Openstack }}
[Global]
auth-url = {{ .DC.Spec.Openstack.AuthURL| quote }}
username = {{ .Cluster.Spec.Cloud.Openstack.Username| quote }}
password = {{ .Cluster.Spec.Cloud.Openstack.Password| quote }}
domain-name= {{ .Cluster.Spec.Cloud.Openstack.Domain| quote }}
tenant-name = {{ .Cluster.Spec.Cloud.Openstack.Tenant| quote }}
region = {{ .DC.Spec.Openstack.Region| quote }}

[BlockStorage]
trust-device-path = false
bs-version = "v2"
{{- if semverCompare ">=1.9.*" .Cluster.Spec.Version }}
ignore-volume-az = {{ .DC.Spec.Openstack.IgnoreVolumeAZ }}
{{- end }}

[LoadBalancer]
{{- if semverCompare "~1.9.10 || ~1.10.6 || ~1.11.1 || >=1.12.*" .Cluster.Spec.Version }}
manage-security-groups = true
{{- end }}
{{- end }}
{{- if .Cluster.Spec.Cloud.Azure}}
{
  "cloud": "AZUREPUBLICCLOUD",
  "tenantId": {{ .Cluster.Spec.Cloud.Azure.TenantID| quote }},
  "subscriptionId": {{ .Cluster.Spec.Cloud.Azure.SubscriptionID| quote }},
  "aadClientId": {{ .Cluster.Spec.Cloud.Azure.ClientID| quote }},
  "aadClientSecret": {{ .Cluster.Spec.Cloud.Azure.ClientSecret| quote }},

  "resourceGroup": {{ .Cluster.Spec.Cloud.Azure.ResourceGroup| quote }},
  "location": {{ .DC.Spec.Azure.Location| quote }},
  "vnetName": {{ .Cluster.Spec.Cloud.Azure.VNetName| quote }},
  "vnetResourceGroup": {{ .Cluster.Spec.Cloud.Azure.ResourceGroup| quote }},
  "subnetName": {{ .Cluster.Spec.Cloud.Azure.SubnetName| quote }},
  "routeTableName": {{ .Cluster.Spec.Cloud.Azure.RouteTableName| quote }},
  "securityGroupName": {{ .Cluster.Spec.Cloud.Azure.SecurityGroup| quote }},
  "primaryAvailabilitySetName": {{ .Cluster.Spec.Cloud.Azure.AvailabilitySet| quote }},

{{/* Consumed by apiserver and controller-manager */}}
  "useInstanceMetadata": false
}
{{- end }}
{{- if .Cluster.Spec.Cloud.VSphere }}
{{/* Source: https://docs.openshift.com/container-platform/3.7/install_config/configuring_vsphere.html#vsphere-enabling */}}
[Global]
        user = {{ .Cluster.Spec.Cloud.VSphere.Username | quote }}
        password = {{ .Cluster.Spec.Cloud.VSphere.Password | quote }}
        server = {{ .DC.Spec.VSphere.Endpoint | replace "https://" ""  | quote }}
        port = 443
        insecure-flag = {{ if .DC.Spec.VSphere.AllowInsecure }}1{{ else }}0{{ end }}
        datacenter = {{ .DC.Spec.VSphere.Datacenter | quote }}
        datastore = {{ .DC.Spec.VSphere.Datastore | quote }}
        working-dir = "{{ .Cluster.Name }}"
        vm-uuid = "vm-uuid"
[Disk]
    scsicontrollertype = pvscsi
{{- end }}
`
)
