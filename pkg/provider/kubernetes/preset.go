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
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// presetsGetter is a function to retrieve preset list
type presetsGetter = func(userInfo *provider.UserInfo) ([]kubermaticv1.Preset, error)

// LoadPresets loads the custom presets for supported providers
func LoadPresets(yamlContent []byte) (*kubermaticv1.PresetList, error) {
	s := struct {
		Presets *kubermaticv1.PresetList `json:"presets"`
	}{}

	err := yaml.UnmarshalStrict(yamlContent, &s)
	if err != nil {
		return nil, err
	}

	return s.Presets, nil
}

// LoadPresetsFromFile loads the custom presets for supported providers
func LoadPresetsFromFile(path string) (*kubermaticv1.PresetList, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	return LoadPresets(bytes)
}

func presetsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, presetsFile string, dynamicPresets bool) (presetsGetter, error) {
	if dynamicPresets {
		return func(userInfo *provider.UserInfo) ([]kubermaticv1.Preset, error) {
			presetList := &kubermaticv1.PresetList{}
			if err := client.List(ctx, presetList); err != nil {
				return nil, fmt.Errorf("failed to get presets %v", err)
			}
			return filterOutPresets(userInfo, presetList)
		}, nil
	}
	var presets *kubermaticv1.PresetList
	var err error

	if presetsFile != "" {
		presets, err = LoadPresetsFromFile(presetsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load presets from %s: %v", presetsFile, err)
		}
	}

	if presets == nil {
		presets = &kubermaticv1.PresetList{Items: []kubermaticv1.Preset{}}
	}

	return func(userInfo *provider.UserInfo) ([]kubermaticv1.Preset, error) {
		return filterOutPresets(userInfo, presets)
	}, nil
}

// PresetsProvider is a object to handle presets from a predefined config
type PresetsProvider struct {
	presetsGetter presetsGetter
}

func NewPresetsProvider(ctx context.Context, client ctrlruntimeclient.Client, presetsFile string, dynamicPresets bool) (*PresetsProvider, error) {
	presetsGetter, err := presetsGetterFactory(ctx, client, presetsFile, dynamicPresets)
	if err != nil {
		return nil, err
	}
	return &PresetsProvider{presetsGetter: presetsGetter}, nil
}

// GetPresets returns presets which belong to the specific email group and for all users
func (m *PresetsProvider) GetPresets(userInfo *provider.UserInfo) ([]kubermaticv1.Preset, error) {
	return m.presetsGetter(userInfo)
}

// GetPreset returns preset with the name which belong to the specific email group
func (m *PresetsProvider) GetPreset(userInfo *provider.UserInfo, name string) (*kubermaticv1.Preset, error) {
	presets, err := m.presetsGetter(userInfo)
	if err != nil {
		return nil, err
	}
	for _, preset := range presets {
		if preset.Name == name {
			return &preset, nil
		}
	}

	return nil, fmt.Errorf("missing preset '%s' for the user '%s'", name, userInfo.Email)
}

func filterOutPresets(userInfo *provider.UserInfo, list *kubermaticv1.PresetList) ([]kubermaticv1.Preset, error) {
	if list == nil {
		return nil, fmt.Errorf("the preset list can not be nil")
	}
	var presetList []kubermaticv1.Preset

	for _, preset := range list.Items {
		requiredEmailDomain := preset.Spec.RequiredEmailDomain
		// find preset for specific email domain
		if requiredEmailDomain != "" {
			userDomain := strings.Split(userInfo.Email, "@")
			if len(userDomain) == 2 && strings.EqualFold(userDomain[1], requiredEmailDomain) {
				presetList = append(presetList, preset)
			}
		} else {
			// find preset for "all" without RequiredEmailDomain field
			presetList = append(presetList, preset)
		}
	}
	return presetList, nil
}

func (m *PresetsProvider) SetCloudCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error) {

	if cloud.VSphere != nil {
		return m.setVsphereCredentials(userInfo, presetName, cloud)
	}
	if cloud.Openstack != nil {
		return m.setOpenStackCredentials(userInfo, presetName, cloud, dc)
	}
	if cloud.Azure != nil {
		return m.setAzureCredentials(userInfo, presetName, cloud)
	}
	if cloud.Digitalocean != nil {
		return m.setDigitalOceanCredentials(userInfo, presetName, cloud)
	}
	if cloud.Packet != nil {
		return m.setPacketCredentials(userInfo, presetName, cloud)
	}
	if cloud.Hetzner != nil {
		return m.setHetznerCredentials(userInfo, presetName, cloud)
	}
	if cloud.AWS != nil {
		return m.setAWSCredentials(userInfo, presetName, cloud)
	}
	if cloud.GCP != nil {
		return m.setGCPCredentials(userInfo, presetName, cloud)
	}
	if cloud.Fake != nil {
		return m.setFakeCredentials(userInfo, presetName, cloud)
	}
	if cloud.Kubevirt != nil {
		return m.setKubevirtCredentials(userInfo, presetName, cloud)
	}
	if cloud.Alibaba != nil {
		return m.setAlibabaCredentials(userInfo, presetName, cloud)
	}

	return nil, fmt.Errorf("can not find provider to set credentials")
}

func emptyCredentialError(preset, provider string) error {
	return fmt.Errorf("the preset %s doesn't contain credential for %s provider", preset, provider)
}

func (m *PresetsProvider) setFakeCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, err
	}
	if preset.Spec.Fake == nil {
		return nil, emptyCredentialError(presetName, "Fake")
	}

	cloud.Fake.Token = preset.Spec.Fake.Token
	return &cloud, nil

}

func (m *PresetsProvider) setKubevirtCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, err
	}

	if preset.Spec.Kubevirt == nil {
		return nil, emptyCredentialError(presetName, "Kubevirt")
	}

	cloud.Kubevirt.Kubeconfig = preset.Spec.Kubevirt.Kubeconfig
	return &cloud, nil
}

func (m *PresetsProvider) setGCPCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, err
	}

	if preset.Spec.GCP == nil {
		return nil, emptyCredentialError(presetName, "GCP")
	}

	credentials := preset.Spec.GCP
	cloud.GCP.ServiceAccount = credentials.ServiceAccount
	cloud.GCP.Network = credentials.Network
	cloud.GCP.Subnetwork = credentials.Subnetwork
	return &cloud, nil

}

func (m *PresetsProvider) setAWSCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, err
	}
	if preset.Spec.AWS == nil {
		return nil, emptyCredentialError(presetName, "AWS")
	}

	credentials := preset.Spec.AWS

	cloud.AWS.AccessKeyID = credentials.AccessKeyID
	cloud.AWS.SecretAccessKey = credentials.SecretAccessKey

	cloud.AWS.InstanceProfileName = credentials.InstanceProfileName
	cloud.AWS.RouteTableID = credentials.RouteTableID
	cloud.AWS.SecurityGroupID = credentials.SecurityGroupID
	cloud.AWS.VPCID = credentials.VPCID
	cloud.AWS.ControlPlaneRoleARN = credentials.ControlPlaneRoleARN
	return &cloud, nil
}

func (m *PresetsProvider) setHetznerCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, err
	}
	if preset.Spec.Hetzner == nil {
		return nil, emptyCredentialError(presetName, "Hetzner")
	}

	cloud.Hetzner.Token = preset.Spec.Hetzner.Token
	return &cloud, nil

}

func (m *PresetsProvider) setPacketCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, err
	}
	if preset.Spec.Packet == nil {
		return nil, emptyCredentialError(presetName, "Packet")
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

func (m *PresetsProvider) setDigitalOceanCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, err
	}
	if preset.Spec.Digitalocean == nil {
		return nil, emptyCredentialError(presetName, "Digitalocean")
	}

	cloud.Digitalocean.Token = preset.Spec.Digitalocean.Token
	return &cloud, nil

}

func (m *PresetsProvider) setAzureCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, err
	}
	if preset.Spec.Azure == nil {
		return nil, emptyCredentialError(presetName, "Azure")
	}

	credentials := preset.Spec.Azure
	cloud.Azure.TenantID = credentials.TenantID
	cloud.Azure.ClientSecret = credentials.ClientSecret
	cloud.Azure.ClientID = credentials.ClientID
	cloud.Azure.SubscriptionID = credentials.SubscriptionID

	cloud.Azure.ResourceGroup = credentials.ResourceGroup
	cloud.Azure.RouteTableName = credentials.RouteTableName
	cloud.Azure.SecurityGroup = credentials.SecurityGroup
	cloud.Azure.SubnetName = credentials.SubnetName
	cloud.Azure.VNetName = credentials.VNetName
	return &cloud, nil

}

func (m *PresetsProvider) setOpenStackCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, err
	}
	if preset.Spec.Openstack == nil {
		return nil, emptyCredentialError(presetName, "Openstack")
	}

	credentials := preset.Spec.Openstack

	cloud.Openstack.Username = credentials.Username
	cloud.Openstack.Password = credentials.Password
	cloud.Openstack.Domain = credentials.Domain
	cloud.Openstack.Tenant = credentials.Tenant
	cloud.Openstack.TenantID = credentials.TenantID

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

func (m *PresetsProvider) setVsphereCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, err
	}
	if preset.Spec.VSphere == nil {
		return nil, emptyCredentialError(presetName, "Vsphere")
	}
	credentials := preset.Spec.VSphere
	cloud.VSphere.Password = credentials.Password
	cloud.VSphere.Username = credentials.Username

	cloud.VSphere.VMNetName = credentials.VMNetName
	cloud.VSphere.Datastore = credentials.Datastore
	cloud.VSphere.DatastoreCluster = credentials.DatastoreCluster
	return &cloud, nil

}

func (m *PresetsProvider) setAlibabaCredentials(userInfo *provider.UserInfo, presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset, err := m.GetPreset(userInfo, presetName)
	if err != nil {
		return nil, err
	}
	if preset.Spec.Alibaba == nil {
		return nil, emptyCredentialError(presetName, "Alibaba")
	}

	credentials := preset.Spec.Alibaba

	cloud.Alibaba.AccessKeyID = credentials.AccessKeyID
	cloud.Alibaba.AccessKeySecret = credentials.AccessKeySecret
	return &cloud, nil
}
