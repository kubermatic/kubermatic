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
		return nil, fmt.Errorf("failed to parse cloud config template: %v", err)
	}
	if err := configTpl.Execute(&configBuffer, data); err != nil {
		return nil, fmt.Errorf("failed to render cloud config template: %v", err)
	}

	cm.Name = resources.CloudConfigConfigMapName
	cm.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	cm.Labels = resources.GetLabels("cloud-config")
	cm.Data = map[string]string{
		"config":              configBuffer.String(),
		FakeVMWareUUIDKeyName: fakeVMWareUUID,
	}

	return cm, nil
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
KubernetesClusterTag={{ .Cluster.Name }}
disablesecuritygroupingress=false
SubnetID={{ .Cluster.Spec.Cloud.AWS.SubnetID }}
RouteTableID={{ .Cluster.Spec.Cloud.AWS.RouteTableID }}
disablestrictzonecheck=true
{{- end }}
{{- if .Cluster.Spec.Cloud.Openstack }}
[Global]
auth-url = "{{ .Cluster.Spec.Cloud.Openstack.AuthURL }}"
username = "{{ .Cluster.Spec.Cloud.Openstack.Username }}"
password = "{{ .Cluster.Spec.Cloud.Openstack.Password }}"
domain-name= "{{ .Cluster.Spec.Cloud.Openstack.Domain }}"
tenant-name = "{{ .Cluster.Spec.Cloud.Openstack.Tenant }}"
region = "{{ .Cluster.Spec.Cloud.Openstack.Region }}"

[BlockStorage]
trust-device-path = false
bs-version = "v2"
{{- if eq (substr 0 3 .Cluster.Spec.Version) "1.9" }}
ignore-volume-az = {{ .Cluster.Spec.Cloud.Openstack.IgnoreVolumeAZ }}
{{- end }}
{{- if eq (substr 0 4 .Cluster.Spec.Version) "1.10" }}
ignore-volume-az = {{ .Cluster.Spec.Cloud.Openstack.IgnoreVolumeAZ }}
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
  "location": "{{ .Cluster.Spec.Cloud.Azure.Location }}",
  "vnetName": "{{ .Cluster.Spec.Cloud.Azure.VNetName }}",
  "vnetResourceGroup": "{{ .Cluster.Spec.Cloud.Azure.ResourceGroup }}",
  "subnetName": "{{ .Cluster.Spec.Cloud.Azure.SubnetName }}",
  "routeTableName": "{{ .Cluster.Spec.Cloud.Azure.RouteTableName }}",

  "useInstanceMetadata": true
}
{{- end }}
{{- if .Cluster.Spec.Cloud.VSphere }}
{{/* Source: https://docs.openshift.com/container-platform/3.7/install_config/configuring_vsphere.html#vsphere-enabling */}}
[Global]
        user = "{{ .Cluster.Spec.Cloud.VSphere.Username }}"
        password = "{{ .Cluster.Spec.Cloud.VSphere.Password }}"
        server = "{{ .Cluster.Spec.Cloud.VSphere.Endpoint|replace "https://" "" }}"
        port = "443"
        insecure-flag = "{{ if .Cluster.Spec.Cloud.VSphere.AllowInsecure }}1{{ else }}0{{ end }}"
        datacenter = "{{ .Cluster.Spec.Cloud.VSphere.Datacenter }}"
        datastore = "{{ .Cluster.Spec.Cloud.VSphere.Datastore }}"
        working-dir = "{{ .Cluster.Name }}"
        vm-uuid = "vm-uuid"
[Disk]
    scsicontrollertype = pvscsi
{{- end }}
`
)
