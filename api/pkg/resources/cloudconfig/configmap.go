package cloudconfig

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/azure"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/gce"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	corev1 "k8s.io/api/core/v1"
)

const (
	name = "cloud-config"
)

type configMapCreatorData interface {
	DC() *kubermaticv1.Datacenter
	Cluster() *kubermaticv1.Cluster
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
}

// ConfigMapCreator returns a function to create the ConfigMap containing the cloud-config
func ConfigMapCreator(data configMapCreatorData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.CloudConfigConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, err
			}

			cloudConfig, err := CloudConfig(data.Cluster(), data.DC(), credentials)
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
func CloudConfig(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter, credentials resources.Credentials) (cloudConfig string, err error) {
	cloud := cluster.Spec.Cloud

	switch {
	case cloud.AWS != nil:
		awsCloudConfig := &aws.CloudConfig{
			// Dummy AZ, so that K8S can extract the region from it.
			// https://github.com/kubernetes/kubernetes/blob/v1.15.0/staging/src/k8s.io/legacy-cloud-providers/aws/aws.go#L1199
			// https://github.com/kubernetes/kubernetes/blob/v1.15.0/staging/src/k8s.io/legacy-cloud-providers/aws/aws.go#L1174
			Global: aws.GlobalOpts{
				Zone:                        dc.Spec.AWS.Region + "x",
				VPC:                         cloud.AWS.VPCID,
				KubernetesClusterID:         cluster.Name,
				DisableSecurityGroupIngress: false,
				RouteTableID:                cloud.AWS.RouteTableID,
				DisableStrictZoneCheck:      true,
				RoleARN:                     cloud.AWS.ControlPlaneRoleARN,
			},
		}
		cloudConfig, err = aws.CloudConfigToString(awsCloudConfig)
		if err != nil {
			return cloudConfig, err
		}

	case cloud.Azure != nil:
		vnetResourceGroup := cloud.Azure.ResourceGroup
		azureCloudConfig := &azure.CloudConfig{
			Cloud:                      "AZUREPUBLICCLOUD",
			TenantID:                   credentials.Azure.TenantID,
			SubscriptionID:             credentials.Azure.SubscriptionID,
			AADClientID:                credentials.Azure.ClientID,
			AADClientSecret:            credentials.Azure.ClientSecret,
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

	case cloud.Openstack != nil:
		manageSecurityGroups := dc.Spec.Openstack.ManageSecurityGroups
		trustDevicePath := dc.Spec.Openstack.TrustDevicePath
		openstackCloudConfig := &openstack.CloudConfig{
			Global: openstack.GlobalOpts{
				AuthURL:    dc.Spec.Openstack.AuthURL,
				Username:   cloud.Openstack.Username,
				Password:   cloud.Openstack.Password,
				DomainName: cloud.Openstack.Domain,
				TenantName: cloud.Openstack.Tenant,
				TenantID:   cloud.Openstack.TenantID,
				Region:     dc.Spec.Openstack.Region,
			},
			BlockStorage: openstack.BlockStorageOpts{
				BSVersion:       "auto",
				TrustDevicePath: trustDevicePath != nil && *trustDevicePath,
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

	case cloud.VSphere != nil:
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

	case cloud.GCP != nil:
		b, err := base64.StdEncoding.DecodeString(credentials.GCP.ServiceAccount)
		if err != nil {
			return "", fmt.Errorf("error decoding service account: %v", err)
		}
		sam := map[string]string{}
		err = json.Unmarshal(b, &sam)
		if err != nil {
			return "", fmt.Errorf("failed unmarshalling service account: %v", err)
		}
		projectID := sam["project_id"]
		if projectID == "" {
			return "", errors.New("empty project_id")
		}

		tag := fmt.Sprintf("kubernetes-cluster-%s", cluster.Name)
		localZone := dc.Spec.GCP.Region + "-" + dc.Spec.GCP.ZoneSuffixes[0]
		var multizone bool
		if len(dc.Spec.GCP.ZoneSuffixes) > 1 {
			multizone = true
		}
		if cloud.GCP.Network == "" {
			cloud.GCP.Network = "global/networks/default"
		}

		gcpCloudConfig := &gce.CloudConfig{
			Global: gce.GlobalOpts{
				ProjectID:      projectID,
				LocalZone:      localZone,
				MultiZone:      multizone,
				Regional:       dc.Spec.GCP.Regional,
				NetworkName:    cloud.GCP.Network,
				SubnetworkName: cloud.GCP.Subnetwork,
				TokenURL:       "nil",
				NodeTags:       []string{tag},
			},
		}
		cloudConfig, err = gcpCloudConfig.AsString()
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
