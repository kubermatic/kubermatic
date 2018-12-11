package cloudconfig

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/azure"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere"
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

		var cloudConfig string
		var err error
		cloud := data.Cluster().Spec.Cloud
		dc := data.DC()
		if cloud.AWS != nil {
			cloudConfig, err = CloudConfig(data)
			if err != nil {
				return nil, err
			}
			awsCloudConfig := &aws.Config{
				Global: aws.GlobalOpts{
					Zone:                        cloud.AWS.AvailabilityZone,
					VPC:                         cloud.AWS.VPCID,
					KubernetesClusterID:         data.Cluster().Name,
					DisableSecurityGroupIngress: false,
					SubnetID:                    cloud.AWS.SubnetID,
					RouteTableID:                cloud.AWS.RouteTableID,
					DisableStrictZoneCheck:      true,
				},
			}
		} else if cloud.Azure != nil {
			//TODO: We need SecurityGroupName, PrimaryAvailabilitySetName
			// and VnetResourceGroup, this requires updating the machine-controller
			// with a version that includes kubermatic/machine-controller#411
			azureCloudConfig := &azure.CloudConfig{
				Cloud:               "AZUREPUBLICCLOUD",
				TenantID:            cloud.Azure.TenantID,
				SubscriptionID:      cloud.Azure.SubscriptionID,
				AADClientID:         cloud.Azure.ClientID,
				AADClientSecret:     cloud.Azure.ClientSecret,
				ResourceGroup:       cloud.Azure.ResourceGroup,
				Location:            dc.Spec.Azure.Location,
				VNetName:            cloud.Azure.VNetName,
				SubnetName:          cloud.Azure.SubnetName,
				RouteTableName:      cloud.Azure.RouteTableName,
				UseInstanceMetadata: false,
			}
			cloudConfig, err = azure.CloudConfigToString(azureCloudConfig)
			if err != nil {
				return nil, err
			}
		} else if cloud.Openstack != nil {
			openstackCloudConfig := &openstack.CloudConfig{
				Global: openstack.GlobalOpts{
					AuthURL:    dc.Spec.Openstack.AuthURL,
					Username:   cloud.Openstack.Username,
					Password:   cloud.Openstack.Password,
					DomainName: cloud.Openstack.Domain,
					TenantName: cloud.Openstack.Tenant,
					Region:     dc.Spec.Openstack.Region,
				},
				BlockStorage: openstack.BlockStorageOpts{
					BSVersion:       "v2",
					TrustDevicePath: false,
				},
			}
			//TODO: Wait until the cloud-config struct in the machine-controller
			// for Openstack has support for switching the loadbalancers
			// manage-security-groups on/off based on version, then add it here
			cloudConfig, err = openstack.CloudConfigToString(openstackCloudConfig)
			if err != nil {
				return nil, err
			}
		} else if cloud.VSphere != nil {
			vsphereCloudConfig := &vsphere.CloudConfig{
				Global: vsphere.GlobalOpts{
					User:         cloud.VSphere.Username,
					Password:     cloud.VSphere.Password,
					InsecureFlag: dc.Spec.VSphere.AllowInsecure,
					VCenterPort:  "443",
				},
				Disk: vsphere.DiskOpts{
					SCSIControllerType: "pvscsi",
				},
				Workspace: vsphere.WorkspaceOpts{
					VCenterIP:        strings.Replace(dc.Spec.VSphere.Endpoint, "https://", "", -1),
					Datacenter:       dc.Spec.VSphere.Datacenter,
					DefaultDatastore: dc.Spec.VSphere.Datastore,
					//TODO: Verify this has the same effect as Global.Working-dir
					Folder: data.Cluster().Name,
				},
			}
			cloudConfig, err = vsphere.CloudConfigToString(vsphereCloudConfig)
			if err != nil {
				return nil, err
			}
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
)
