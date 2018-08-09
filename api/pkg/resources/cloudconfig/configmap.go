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

	cloudConfig, err := CloudConfig(data)
	if err != nil {
		return nil, err
	}

	cm.Name = resources.CloudConfigConfigMapName
	cm.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	cm.Labels = resources.BaseAppLabel(name, nil)
	cm.Data = map[string]string{
		"config":              cloudConfig,
		FakeVMWareUUIDKeyName: fakeVMWareUUID,
	}

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
zone={{ .Cluster.Spec.Cloud.AWS.AvailabilityZone }}
VPC={{ .Cluster.Spec.Cloud.AWS.VPCID }}
KubernetesClusterID={{ .Cluster.Name }}
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
  "tenantId": "{{ .Cluster.Spec.Cloud.Azure.TenantID }}",
  "subscriptionId": "{{ .Cluster.Spec.Cloud.Azure.SubscriptionID }}",
  "aadClientId": "{{ .Cluster.Spec.Cloud.Azure.ClientID }}",
  "aadClientSecret": "{{ .Cluster.Spec.Cloud.Azure.ClientSecret }}",

  "resourceGroup": "{{ .Cluster.Spec.Cloud.Azure.ResourceGroup }}",
  "location": "{{ .DC.Spec.Azure.Location }}",
  "vnetName": "{{ .Cluster.Spec.Cloud.Azure.VNetName }}",
  "vnetResourceGroup": "{{ .Cluster.Spec.Cloud.Azure.ResourceGroup }}",
  "subnetName": "{{ .Cluster.Spec.Cloud.Azure.SubnetName }}",
  "routeTableName": "{{ .Cluster.Spec.Cloud.Azure.RouteTableName }}",
  "securityGroupName": "{{ .Cluster.Spec.Cloud.Azure.SecurityGroup }}",
  "primaryAvailabilitySetName": "{{ .Cluster.Spec.Cloud.Azure.AvailabilitySet }}",

{{/* Consumed by apiserver and controller-manager */}}
  "useInstanceMetadata": false
}
{{- end }}
{{- if .Cluster.Spec.Cloud.VSphere }}
{{/* Source: https://docs.openshift.com/container-platform/3.7/install_config/configuring_vsphere.html#vsphere-enabling */}}
[Global]
        user = "{{ .Cluster.Spec.Cloud.VSphere.Username }}"
        password = "{{ .Cluster.Spec.Cloud.VSphere.Password }}"
        server = "{{ .DC.Spec.VSphere.Endpoint|replace "https://" "" }}"
        port = "443"
        insecure-flag = "{{ if .DC.Spec.VSphere.AllowInsecure }}1{{ else }}0{{ end }}"
        datacenter = "{{ .DC.Spec.VSphere.Datacenter }}"
        datastore = "{{ .DC.Spec.VSphere.Datastore }}"
        working-dir = "{{ .Cluster.Name }}"
        vm-uuid = "vm-uuid"
[Disk]
    scsicontrollertype = pvscsi
{{- end }}
`
)
