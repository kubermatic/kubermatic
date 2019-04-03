package cloudconfig

import (
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/azure"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere"

	corev1 "k8s.io/api/core/v1"
)

const (
	name = "cloud-config"
)

type configMapCreatorData interface {
	DC() *provider.DatacenterMeta
	Cluster() *kubermaticv1.Cluster
}

// ConfigMapCreator returns a function to create the ConfigMap containing the cloud-config
func ConfigMapCreator(data configMapCreatorData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.CloudConfigConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			cloudConfig, err := CloudConfig(data.Cluster(), data.DC())
			if err != nil {
				return nil, fmt.Errorf("failed to create cloud-config: %v", err)
			}

			cm.Labels = resources.BaseAppLabel(name, nil)
			cm.Data["config"] = cloudConfig
			cm.Data[FakeVMWareUUIDKeyName] = fakeVMWareUUID

			return cm, nil
		}
	}
}

// CloudConfig returns the cloud-config for the supplied data
func CloudConfig(cluster *kubermaticv1.Cluster, dc *provider.DatacenterMeta) (cloudConfig string, err error) {
	cloud := cluster.Spec.Cloud
	if cloud.AWS != nil {
		awsCloudConfig := &aws.CloudConfig{
			Global: aws.GlobalOpts{
				Zone:                        cloud.AWS.AvailabilityZone,
				VPC:                         cloud.AWS.VPCID,
				KubernetesClusterID:         cluster.Name,
				DisableSecurityGroupIngress: false,
				SubnetID:                    cloud.AWS.SubnetID,
				RouteTableID:                cloud.AWS.RouteTableID,
				DisableStrictZoneCheck:      true,
			},
		}
		cloudConfig, err = aws.CloudConfigToString(awsCloudConfig)
		if err != nil {
			return cloudConfig, err
		}
	} else if cloud.Azure != nil {
		vnetResourceGroup := cloud.Azure.ResourceGroup
		azureCloudConfig := &azure.CloudConfig{
			Cloud:                      "AZUREPUBLICCLOUD",
			TenantID:                   cloud.Azure.TenantID,
			SubscriptionID:             cloud.Azure.SubscriptionID,
			AADClientID:                cloud.Azure.ClientID,
			AADClientSecret:            cloud.Azure.ClientSecret,
			ResourceGroup:              cloud.Azure.ResourceGroup,
			Location:                   dc.Spec.Azure.Location,
			VNetName:                   cloud.Azure.VNetName,
			SubnetName:                 cloud.Azure.SubnetName,
			RouteTableName:             cloud.Azure.RouteTableName,
			SecurityGroupName:          cloud.Azure.SecurityGroup,
			PrimaryAvailabilitySetName: cloud.Azure.AvailabilitySet,
			VnetResourceGroup:          &vnetResourceGroup,
			UseInstanceMetadata:        false,
		}
		cloudConfig, err = azure.CloudConfigToString(azureCloudConfig)
		if err != nil {
			return cloudConfig, err
		}
	} else if cloud.Openstack != nil {
		manageSecurityGroups := dc.Spec.Openstack.ManageSecurityGroups
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
				IgnoreVolumeAZ:  dc.Spec.Openstack.IgnoreVolumeAZ,
			},
			LoadBalancer: openstack.LoadBalancerOpts{
				ManageSecurityGroups: manageSecurityGroups == nil || *manageSecurityGroups,
			},
			Version: cluster.Spec.Version.String(),
		}
		cloudConfig, err = openstack.CloudConfigToString(openstackCloudConfig)
		if err != nil {
			return cloudConfig, err
		}
	} else if cloud.VSphere != nil {
		vsphereCloudConfig := &vsphere.CloudConfig{
			Global: vsphere.GlobalOpts{
				User:             cloud.VSphere.Username,
				Password:         cloud.VSphere.Password,
				VCenterIP:        strings.Replace(dc.Spec.VSphere.Endpoint, "https://", "", -1),
				VCenterPort:      "443",
				InsecureFlag:     dc.Spec.VSphere.AllowInsecure,
				Datacenter:       dc.Spec.VSphere.Datacenter,
				DefaultDatastore: dc.Spec.VSphere.Datastore,
				WorkingDir:       cluster.Name,
			},
			Workspace: vsphere.WorkspaceOpts{
				// This is redudant with what the Vsphere cloud provider itself does:
				// https://github.com/kubernetes/kubernetes/blob/9d80e7522ab7fc977e40dd6f3b5b16d8ebfdc435/pkg/cloudprovider/providers/vsphere/vsphere.go#L346
				// We do it here because the fields in the "Global" object
				// are marked as deprecated even thought the code checks
				// if they are set and will make the controller-manager crash
				// if they are not - But maybe that will change at some point
				VCenterIP:        strings.Replace(dc.Spec.VSphere.Endpoint, "https://", "", -1),
				Datacenter:       dc.Spec.VSphere.Datacenter,
				Folder:           cluster.Name,
				DefaultDatastore: dc.Spec.VSphere.Datastore,
			},
			Disk: vsphere.DiskOpts{
				SCSIControllerType: "pvscsi",
			},
		}
		cloudConfig, err = vsphere.CloudConfigToString(vsphereCloudConfig)
		if err != nil {
			return cloudConfig, err
		}
	}
	return cloudConfig, err
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
