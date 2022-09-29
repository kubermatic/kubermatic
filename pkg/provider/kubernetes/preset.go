/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubernetes

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/email"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// presetsGetter is a function to retrieve preset list.
type presetsGetter = func(ctx context.Context, userInfo *provider.UserInfo, projectID *string) ([]kubermaticv1.Preset, error)

// presetCreator is a function to create a preset.
type presetCreator = func(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error)

// presetUpdater is a function to update a preset.
type presetUpdater = func(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error)

// presetDeleter is a function to delete a preset.
type presetDeleter = func(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error)

func presetsGetterFactory(client ctrlruntimeclient.Client) (presetsGetter, error) {
	return func(ctx context.Context, userInfo *provider.UserInfo, projectID *string) ([]kubermaticv1.Preset, error) {
		presetList := &kubermaticv1.PresetList{}
		if err := client.List(ctx, presetList); err != nil {
			return nil, fmt.Errorf("failed to get presets: %w", err)
		}
		return filterOutPresets(userInfo, projectID, presetList)
	}, nil
}

func presetCreatorFactory(client ctrlruntimeclient.Client) (presetCreator, error) {
	return func(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error) {
		if err := client.Create(ctx, preset); err != nil {
			return nil, err
		}

		return preset, nil
	}, nil
}

func presetUpdaterFactory(client ctrlruntimeclient.Client) (presetUpdater, error) {
	return func(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error) {
		if err := client.Update(ctx, preset); err != nil {
			return nil, err
		}

		return preset, nil
	}, nil
}

func presetDeleterFactory(client ctrlruntimeclient.Client) (presetDeleter, error) {
	return func(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error) {
		if err := client.Delete(ctx, preset); err != nil {
			return &kubermaticv1.Preset{}, err
		}
		return &kubermaticv1.Preset{}, nil
	}, nil
}

// PresetProvider is a object to handle presets from a predefined config.
type PresetProvider struct {
	getter  presetsGetter
	creator presetCreator
	patcher presetUpdater
	deleter presetDeleter
}

var _ provider.PresetProvider = &PresetProvider{}

func NewPresetProvider(client ctrlruntimeclient.Client) (*PresetProvider, error) {
	getter, err := presetsGetterFactory(client)
	if err != nil {
		return nil, err
	}

	creator, err := presetCreatorFactory(client)
	if err != nil {
		return nil, err
	}

	patcher, err := presetUpdaterFactory(client)
	if err != nil {
		return nil, err
	}

	deleter, err := presetDeleterFactory(client)
	if err != nil {
		return nil, err
	}

	return &PresetProvider{getter, creator, patcher, deleter}, nil
}

func (m *PresetProvider) CreatePreset(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error) {
	return m.creator(ctx, preset)
}

func (m *PresetProvider) UpdatePreset(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error) {
	return m.patcher(ctx, preset)
}

// GetPreset returns all presets which belong to the specific email group or projectID. Passing `nil` for `projectID` means that
// all presets are considered (i.e. no filtering based on project association takes places). This should only be used in administrative
// contexts.
// For code that cannot provide project information and should not have access to project-scoped presets, pass a pointer to an empty string
// (e.g. `pointer.String("")`) instead. This will enable project ID filtering, but no preset with project associations will be returned.
func (m *PresetProvider) GetPresets(ctx context.Context, userInfo *provider.UserInfo, projectID *string) ([]kubermaticv1.Preset, error) {
	return m.getter(ctx, userInfo, projectID)
}

// GetPreset returns a preset based on its name which belong to the specific email group or projectID. Passing `nil` for `projectID` means that
// all presets are considered (i.e. no filtering based on project association takes places). This should only be used in administrative
// contexts.
// For code that cannot provide project information and should not have access to project-scoped presets, pass a pointer to an empty string
// (e.g. `pointer.String("")`) instead. This will enable project ID filtering, but no preset with project associations will be returned.
func (m *PresetProvider) GetPreset(ctx context.Context, userInfo *provider.UserInfo, projectID *string, name string) (*kubermaticv1.Preset, error) {
	presets, err := m.getter(ctx, userInfo, projectID)
	if err != nil {
		return nil, err
	}
	for _, preset := range presets {
		if preset.Name == name {
			return &preset, nil
		}
	}

	return nil, apierrors.NewNotFound(kubermaticv1.Resource("preset"), name)
}

// DeletePreset delete Preset.
func (m *PresetProvider) DeletePreset(ctx context.Context, preset *kubermaticv1.Preset) (*kubermaticv1.Preset, error) {
	return m.deleter(ctx, preset)
}

func filterOutPresets(userInfo *provider.UserInfo, projectID *string, list *kubermaticv1.PresetList) ([]kubermaticv1.Preset, error) {
	if list == nil {
		return nil, fmt.Errorf("the preset list can not be nil")
	}

	var result []kubermaticv1.Preset

	for _, preset := range list.Items {
		// by default, we "match" the project as we might either not be filtering based on projectID
		// or the preset does not have a project association in place.
		matchesProject := true

		if projectID != nil && preset.Spec.Projects != nil && len(preset.Spec.Projects) > 0 {
			// reset matchesProject to ensure we don't return this preset unless we find a matching project limitation
			matchesProject = false

			for _, project := range preset.Spec.Projects {
				if *projectID == project {
					matchesProject = true
					break
				}
			}
		}

		matchesEmail, err := email.MatchesRequirements(userInfo.Email, preset.Spec.RequiredEmails)
		if err != nil {
			return nil, err
		}

		if matchesProject && (matchesEmail || userInfo.IsAdmin) {
			result = append(result, preset)
		}
	}

	return result, nil
}

func (m *PresetProvider) SetCloudCredentials(ctx context.Context, userInfo *provider.UserInfo, projectID *string, presetName string, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(ctx, userInfo, projectID, presetName)
	if err != nil {
		return nil, err
	}

	if cloud.VSphere != nil {
		return m.setVsphereCredentials(preset, cloud, dc)
	}
	if cloud.Openstack != nil {
		return m.setOpenStackCredentials(preset, cloud, dc)
	}
	if cloud.Azure != nil {
		return m.setAzureCredentials(preset, cloud)
	}
	if cloud.Digitalocean != nil {
		return m.setDigitalOceanCredentials(preset, cloud)
	}
	if cloud.Packet != nil {
		return m.setPacketCredentials(preset, cloud)
	}
	if cloud.Hetzner != nil {
		return m.setHetznerCredentials(preset, cloud)
	}
	if cloud.AWS != nil {
		return m.setAWSCredentials(preset, cloud)
	}
	if cloud.GCP != nil {
		return m.setGCPCredentials(preset, cloud)
	}
	if cloud.Fake != nil {
		return m.setFakeCredentials(preset, cloud)
	}
	if cloud.Kubevirt != nil {
		return m.setKubevirtCredentials(preset, cloud)
	}
	if cloud.Alibaba != nil {
		return m.setAlibabaCredentials(preset, cloud)
	}
	if cloud.Anexia != nil {
		return m.setAnexiaCredentials(preset, cloud)
	}
	if cloud.Nutanix != nil {
		return m.setNutanixCredentials(preset, cloud)
	}
	if cloud.VMwareCloudDirector != nil {
		return m.setVMwareCloudDirectorCredentials(preset, cloud)
	}

	return nil, fmt.Errorf("can not find provider to set credentials")
}

func emptyCredentialError(preset, provider string) error {
	return fmt.Errorf("the preset %s doesn't contain credential for %s provider", preset, provider)
}

func (m *PresetProvider) setFakeCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.Fake == nil {
		return nil, emptyCredentialError(preset.Name, "Fake")
	}

	cloud.Fake.Token = preset.Spec.Fake.Token

	return &cloud, nil
}

func (m *PresetProvider) setKubevirtCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.Kubevirt == nil {
		return nil, emptyCredentialError(preset.Name, "Kubevirt")
	}

	cloud.Kubevirt.Kubeconfig = preset.Spec.Kubevirt.Kubeconfig

	return &cloud, nil
}

func (m *PresetProvider) setGCPCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.GCP == nil {
		return nil, emptyCredentialError(preset.Name, "GCP")
	}

	credentials := preset.Spec.GCP
	cloud.GCP.ServiceAccount = credentials.ServiceAccount
	cloud.GCP.Network = credentials.Network
	cloud.GCP.Subnetwork = credentials.Subnetwork

	return &cloud, nil
}

func (m *PresetProvider) setAWSCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.AWS == nil {
		return nil, emptyCredentialError(preset.Name, "AWS")
	}

	credentials := preset.Spec.AWS

	cloud.AWS.AccessKeyID = credentials.AccessKeyID
	cloud.AWS.SecretAccessKey = credentials.SecretAccessKey

	cloud.AWS.AssumeRoleARN = credentials.AssumeRoleARN
	cloud.AWS.AssumeRoleExternalID = credentials.AssumeRoleExternalID

	cloud.AWS.InstanceProfileName = credentials.InstanceProfileName
	cloud.AWS.RouteTableID = credentials.RouteTableID
	cloud.AWS.SecurityGroupID = credentials.SecurityGroupID
	cloud.AWS.VPCID = credentials.VPCID
	cloud.AWS.ControlPlaneRoleARN = credentials.ControlPlaneRoleARN

	return &cloud, nil
}

func (m *PresetProvider) setHetznerCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.Hetzner == nil {
		return nil, emptyCredentialError(preset.Name, "Hetzner")
	}

	cloud.Hetzner.Token = preset.Spec.Hetzner.Token
	cloud.Hetzner.Network = preset.Spec.Hetzner.Network

	return &cloud, nil
}

func (m *PresetProvider) setPacketCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.Packet == nil {
		return nil, emptyCredentialError(preset.Name, "Packet")
	}

	credentials := preset.Spec.Packet
	cloud.Packet.ProjectID = credentials.ProjectID
	cloud.Packet.APIKey = credentials.APIKey

	cloud.Packet.BillingCycle = credentials.BillingCycle
	if len(credentials.BillingCycle) == 0 {
		cloud.Packet.BillingCycle = "hourly"
	}

	return &cloud, nil
}

func (m *PresetProvider) setDigitalOceanCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.Digitalocean == nil {
		return nil, emptyCredentialError(preset.Name, "Digitalocean")
	}

	cloud.Digitalocean.Token = preset.Spec.Digitalocean.Token

	return &cloud, nil
}

func (m *PresetProvider) setAzureCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.Azure == nil {
		return nil, emptyCredentialError(preset.Name, "Azure")
	}

	credentials := preset.Spec.Azure
	cloud.Azure.TenantID = credentials.TenantID
	cloud.Azure.ClientSecret = credentials.ClientSecret
	cloud.Azure.ClientID = credentials.ClientID
	cloud.Azure.SubscriptionID = credentials.SubscriptionID

	cloud.Azure.ResourceGroup = credentials.ResourceGroup
	cloud.Azure.VNetResourceGroup = credentials.VNetResourceGroup
	cloud.Azure.RouteTableName = credentials.RouteTableName
	cloud.Azure.SecurityGroup = credentials.SecurityGroup
	cloud.Azure.SubnetName = credentials.SubnetName
	cloud.Azure.VNetName = credentials.VNetName

	return &cloud, nil
}

func (m *PresetProvider) setOpenStackCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.Openstack == nil {
		return nil, emptyCredentialError(preset.Name, "Openstack")
	}

	credentials := preset.Spec.Openstack

	cloud.Openstack.Username = credentials.Username
	cloud.Openstack.Password = credentials.Password
	cloud.Openstack.Domain = credentials.Domain
	cloud.Openstack.Project = credentials.Project
	cloud.Openstack.ProjectID = credentials.ProjectID

	cloud.Openstack.UseToken = credentials.UseToken

	cloud.Openstack.ApplicationCredentialID = credentials.ApplicationCredentialID
	cloud.Openstack.ApplicationCredentialSecret = credentials.ApplicationCredentialSecret

	cloud.Openstack.SubnetID = credentials.SubnetID
	cloud.Openstack.Network = credentials.Network
	cloud.Openstack.FloatingIPPool = credentials.FloatingIPPool

	if cloud.Openstack.FloatingIPPool == "" && dc.Spec.Openstack != nil && dc.Spec.Openstack.EnforceFloatingIP {
		return nil, fmt.Errorf("preset error, no floating ip pool specified for OpenStack")
	}

	cloud.Openstack.RouterID = credentials.RouterID
	cloud.Openstack.SecurityGroups = credentials.SecurityGroups

	return &cloud, nil
}

func (m *PresetProvider) setVsphereCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.VSphere == nil {
		return nil, emptyCredentialError(preset.Name, "Vsphere")
	}
	credentials := preset.Spec.VSphere
	cloud.VSphere.Password = credentials.Password
	cloud.VSphere.Username = credentials.Username

	cloud.VSphere.VMNetName = credentials.VMNetName
	cloud.VSphere.Datastore = credentials.Datastore
	cloud.VSphere.DatastoreCluster = credentials.DatastoreCluster
	cloud.VSphere.ResourcePool = credentials.ResourcePool
	if cloud.VSphere.StoragePolicy == "" {
		cloud.VSphere.StoragePolicy = dc.Spec.VSphere.DefaultStoragePolicy
	}

	return &cloud, nil
}

func (m *PresetProvider) setAlibabaCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.Alibaba == nil {
		return nil, emptyCredentialError(preset.Name, "Alibaba")
	}

	credentials := preset.Spec.Alibaba

	cloud.Alibaba.AccessKeyID = credentials.AccessKeyID
	cloud.Alibaba.AccessKeySecret = credentials.AccessKeySecret
	return &cloud, nil
}

func (m *PresetProvider) setAnexiaCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.Anexia == nil {
		return nil, emptyCredentialError(preset.Name, "Anexia")
	}

	cloud.Anexia.Token = preset.Spec.Anexia.Token

	return &cloud, nil
}

func (m *PresetProvider) setNutanixCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.Nutanix == nil {
		return nil, emptyCredentialError(preset.Name, "Nutanix")
	}

	cloud.Nutanix.Username = preset.Spec.Nutanix.Username
	cloud.Nutanix.Password = preset.Spec.Nutanix.Password

	if proxyURL := preset.Spec.Nutanix.ProxyURL; proxyURL != "" {
		cloud.Nutanix.ProxyURL = proxyURL
	}

	if clusterName := preset.Spec.Nutanix.ClusterName; clusterName != "" {
		cloud.Nutanix.ClusterName = clusterName
	}

	if projectName := preset.Spec.Nutanix.ProjectName; projectName != "" {
		cloud.Nutanix.ProjectName = projectName
	}

	if preset.Spec.Nutanix.CSIUsername != "" && preset.Spec.Nutanix.CSIPassword != "" && preset.Spec.Nutanix.CSIEndpoint != "" {
		cloud.Nutanix.CSI = &kubermaticv1.NutanixCSIConfig{
			Username: preset.Spec.Nutanix.CSIUsername,
			Password: preset.Spec.Nutanix.CSIPassword,
			Endpoint: preset.Spec.Nutanix.CSIEndpoint,
			Port:     preset.Spec.Nutanix.CSIPort,
		}
	}

	return &cloud, nil
}

func (m *PresetProvider) setVMwareCloudDirectorCredentials(preset *kubermaticv1.Preset, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	if preset.Spec.VMwareCloudDirector == nil {
		return nil, emptyCredentialError(preset.Name, "VMware Cloud Director")
	}

	credentials := preset.Spec.VMwareCloudDirector
	cloud.VMwareCloudDirector.Username = credentials.Username
	cloud.VMwareCloudDirector.Password = credentials.Password
	cloud.VMwareCloudDirector.Organization = credentials.Organization
	cloud.VMwareCloudDirector.VDC = credentials.VDC
	cloud.VMwareCloudDirector.OVDCNetwork = credentials.OVDCNetwork

	return &cloud, nil
}
