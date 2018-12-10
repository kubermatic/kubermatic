package cloudconfig

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/machine-controller/pkg/ini"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	name = "cloud-config"
)

// ConfigMapCreator returns a function to create the ConfigMap containing the cloud-config
func ConfigMapCreator(data resources.ConfigMapDataProvider) resources.ConfigMapCreator {
	return func(existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
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
}

// CloudConfig returns the cloud-config for the supplied data
func CloudConfig(data resources.ConfigMapDataProvider) (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["iniEscape"] = ini.Escape

	configBuffer := bytes.Buffer{}
	configTpl, err := template.New("base").Funcs(funcMap).Parse(config)
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
zone={{ .Cluster.Spec.Cloud.AWS.AvailabilityZone | iniEscape }}
VPC={{ .Cluster.Spec.Cloud.AWS.VPCID | iniEscape }}
KubernetesClusterID={{ .Cluster.Name | iniEscape }}
disablesecuritygroupingress=false
SubnetID={{ .Cluster.Spec.Cloud.AWS.SubnetID | iniEscape }}
RouteTableID={{ .Cluster.Spec.Cloud.AWS.RouteTableID | iniEscape }}
disablestrictzonecheck=true
{{- end }}
{{- if .Cluster.Spec.Cloud.Openstack }}
[Global]
auth-url = {{ .DC.Spec.Openstack.AuthURL | iniEscape }}
username = {{ .Cluster.Spec.Cloud.Openstack.Username | iniEscape }}
password = {{ .Cluster.Spec.Cloud.Openstack.Password | iniEscape }}
domain-name= {{ .Cluster.Spec.Cloud.Openstack.Domain | iniEscape }}
tenant-name = {{ .Cluster.Spec.Cloud.Openstack.Tenant | iniEscape }}
region = {{ .DC.Spec.Openstack.Region |iniEscape }}

[BlockStorage]
trust-device-path = false
bs-version = "v2"
{{- if semverCompare ">=1.9.*" .ClusterVersion }}
ignore-volume-az = {{ .DC.Spec.Openstack.IgnoreVolumeAZ| iniEscape  }}
{{- end }}

[LoadBalancer]
{{- if semverCompare "~1.9.10 || ~1.10.6 || ~1.11.1 || >=1.12.*" .ClusterVersion }}
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
        user = {{ .Cluster.Spec.Cloud.VSphere.Username | iniEscape }}
        password = {{ .Cluster.Spec.Cloud.VSphere.Password | iniEscape }}
        server = {{ .DC.Spec.VSphere.Endpoint|replace "https://" "" | iniEscape }}
        port = "443"
        insecure-flag = {{ if .DC.Spec.VSphere.AllowInsecure }}1{{ else }}0{{ end }}
        datacenter = {{ .DC.Spec.VSphere.Datacenter | iniEscape}}
        datastore = {{ .DC.Spec.VSphere.Datastore | iniEscape }}
        working-dir = {{ .Cluster.Name | iniEscape }}
[Disk]
    scsicontrollertype = pvscsi
{{- end }}
`
)
