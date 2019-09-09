package presets

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"sigs.k8s.io/yaml"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

// loadPresets loads the custom presets for supported providers
func loadPresets(path string) (*kubermaticv1.PresetList, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	s := struct {
		Presets *kubermaticv1.PresetList `json:"presets"`
	}{}

	err = yaml.UnmarshalStrict(bytes, &s)
	if err != nil {
		return nil, err
	}

	return s.Presets, nil
}

// Manager is a object to handle presets from a predefined config
type Manager struct {
	presets *kubermaticv1.PresetList
}

func New() *Manager {
	return &Manager{presets: &kubermaticv1.PresetList{}}
}

func NewWithPresets(presets *kubermaticv1.PresetList) *Manager {
	return &Manager{presets: presets}
}

// NewFromFile returns a instance of manager with the credentials loaded from the given paths
func NewFromFile(credentialsFilename string) (*Manager, error) {
	var presets *kubermaticv1.PresetList
	var err error

	if len(credentialsFilename) > 0 {
		presets, err = loadPresets(credentialsFilename)
		if err != nil {
			return nil, fmt.Errorf("failed to load presets from %s: %v", credentialsFilename, err)
		}
	}
	if presets == nil {
		presets = &kubermaticv1.PresetList{}

	}
	return &Manager{presets: presets}, nil
}

func (m *Manager) GetPreset(userInfo provider.UserInfo) *kubermaticv1.Preset {
	for _, preset := range m.presets.Items {
		emialDomain := preset.Spec.RequiredEmailDomain
		domain := strings.Split(userInfo.Email, "@")
		if len(domain) == 2 {
			if strings.EqualFold(domain[1], emialDomain) {
				return &preset
			}
		}
	}
	return emptyPreset()
}

func (m *Manager) SetCloudCredentials(userInfo provider.UserInfo, credentialName string, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error) {

	if cloud.VSphere != nil {
		return m.setVsphereCredentials(userInfo, credentialName, cloud)
	}
	if cloud.Openstack != nil {
		return m.setOpenStackCredentials(userInfo, credentialName, cloud, dc)
	}
	if cloud.Azure != nil {
		return m.setAzureCredentials(userInfo, credentialName, cloud)
	}
	if cloud.Digitalocean != nil {
		return m.setDigitalOceanCredentials(userInfo, credentialName, cloud)
	}
	if cloud.Packet != nil {
		return m.setPacketCredentials(userInfo, credentialName, cloud)
	}
	if cloud.Hetzner != nil {
		return m.setHetznerCredentials(userInfo, credentialName, cloud)
	}
	if cloud.AWS != nil {
		return m.setAWSCredentials(userInfo, credentialName, cloud)
	}
	if cloud.GCP != nil {
		return m.setGCPCredentials(userInfo, credentialName, cloud)
	}
	if cloud.Fake != nil {
		return m.setFakeCredentials(userInfo, credentialName, cloud)
	}
	if cloud.Kubevirt != nil {
		return m.setKubevirtCredentials(userInfo, credentialName, cloud)
	}

	return nil, fmt.Errorf("can not find provider to set credentials")
}

func emptyCredentialListError(provider string) error {
	return fmt.Errorf("can not find any credential for %s provider", provider)
}

func noCredentialError(credentialName string) error {
	return fmt.Errorf("can not find %s credential", credentialName)
}

func (m *Manager) setFakeCredentials(userInfo provider.UserInfo, credentialName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(userInfo)
	if preset.Spec.Fake.Credentials == nil {
		return nil, emptyCredentialListError("Fake")
	}
	for _, credential := range preset.Spec.Fake.Credentials {
		if credentialName == credential.Name {
			cloud.Fake.Token = credential.Token
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setKubevirtCredentials(userInfo provider.UserInfo, credentialName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(userInfo)
	if preset.Spec.Kubevirt.Credentials == nil {
		return nil, emptyCredentialListError("Kubevirt")
	}
	for _, credential := range preset.Spec.Kubevirt.Credentials {
		if credentialName == credential.Name {

			cloud.Kubevirt.Kubeconfig = credential.Kubeconfig
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setGCPCredentials(userInfo provider.UserInfo, credentialName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(userInfo)
	if preset.Spec.GCP.Credentials == nil {
		return nil, emptyCredentialListError("GCP")
	}
	for _, credential := range preset.Spec.GCP.Credentials {
		if credentialName == credential.Name {
			cloud.GCP.ServiceAccount = credential.ServiceAccount

			cloud.GCP.Network = credential.Network
			cloud.GCP.Subnetwork = credential.Subnetwork
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setAWSCredentials(userInfo provider.UserInfo, credentialName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(userInfo)
	if preset.Spec.AWS.Credentials == nil {
		return nil, emptyCredentialListError("AWS")
	}
	for _, credential := range preset.Spec.AWS.Credentials {
		if credentialName == credential.Name {
			cloud.AWS.AccessKeyID = credential.AccessKeyID
			cloud.AWS.SecretAccessKey = credential.SecretAccessKey

			cloud.AWS.InstanceProfileName = credential.InstanceProfileName
			cloud.AWS.RouteTableID = credential.RouteTableID
			cloud.AWS.SecurityGroupID = credential.SecurityGroupID
			cloud.AWS.VPCID = credential.VPCID
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setHetznerCredentials(userInfo provider.UserInfo, credentialName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(userInfo)
	if preset.Spec.Hetzner.Credentials == nil {
		return nil, emptyCredentialListError("Hetzner")
	}
	for _, credential := range preset.Spec.Hetzner.Credentials {
		if credentialName == credential.Name {
			cloud.Hetzner.Token = credential.Token
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setPacketCredentials(userInfo provider.UserInfo, credentialName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(userInfo)
	if preset.Spec.Packet.Credentials == nil {
		return nil, emptyCredentialListError("Packet")
	}
	for _, credential := range preset.Spec.Packet.Credentials {
		if credentialName == credential.Name {
			cloud.Packet.ProjectID = credential.ProjectID
			cloud.Packet.APIKey = credential.APIKey

			cloud.Packet.BillingCycle = credential.BillingCycle
			if len(credential.BillingCycle) == 0 {
				cloud.Packet.BillingCycle = "hourly"
			}

			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setDigitalOceanCredentials(userInfo provider.UserInfo, credentialName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(userInfo)
	if preset.Spec.Digitalocean.Credentials == nil {
		return nil, emptyCredentialListError("Digitalocean")
	}
	for _, credential := range preset.Spec.Digitalocean.Credentials {
		if credentialName == credential.Name {
			cloud.Digitalocean.Token = credential.Token
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setAzureCredentials(userInfo provider.UserInfo, credentialName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(userInfo)
	if preset.Spec.Azure.Credentials == nil {
		return nil, emptyCredentialListError("Azure")
	}
	for _, credential := range preset.Spec.Azure.Credentials {
		if credentialName == credential.Name {
			cloud.Azure.TenantID = credential.TenantID
			cloud.Azure.ClientSecret = credential.ClientSecret
			cloud.Azure.ClientID = credential.ClientID
			cloud.Azure.SubscriptionID = credential.SubscriptionID

			cloud.Azure.ResourceGroup = credential.ResourceGroup
			cloud.Azure.RouteTableName = credential.RouteTableName
			cloud.Azure.SecurityGroup = credential.SecurityGroup
			cloud.Azure.SubnetName = credential.SubnetName
			cloud.Azure.VNetName = credential.VNetName
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setOpenStackCredentials(userInfo provider.UserInfo, credentialName string, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(userInfo)
	if preset.Spec.Openstack.Credentials == nil {
		return nil, emptyCredentialListError("Openstack")
	}
	for _, credential := range preset.Spec.Openstack.Credentials {
		if credentialName == credential.Name {
			cloud.Openstack.Username = credential.Username
			cloud.Openstack.Password = credential.Password
			cloud.Openstack.Domain = credential.Domain
			cloud.Openstack.Tenant = credential.Tenant
			cloud.Openstack.TenantID = credential.TenantID

			cloud.Openstack.SubnetID = credential.SubnetID
			cloud.Openstack.Network = credential.Network
			cloud.Openstack.FloatingIPPool = credential.FloatingIPPool

			if cloud.Openstack.FloatingIPPool == "" && dc.Spec.Openstack != nil && dc.Spec.Openstack.EnforceFloatingIP {
				return nil, fmt.Errorf("preset error, no floating ip pool specified for OpenStack")
			}

			cloud.Openstack.RouterID = credential.RouterID
			cloud.Openstack.SecurityGroups = credential.SecurityGroups
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setVsphereCredentials(userInfo provider.UserInfo, credentialName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(userInfo)
	if preset.Spec.VSphere.Credentials == nil {
		return nil, emptyCredentialListError("Vsphere")
	}
	for _, credential := range preset.Spec.VSphere.Credentials {
		if credentialName == credential.Name {
			cloud.VSphere.Password = credential.Password
			cloud.VSphere.Username = credential.Username

			cloud.VSphere.VMNetName = credential.VMNetName
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func emptyPreset() *kubermaticv1.Preset {
	return &kubermaticv1.Preset{
		Spec: kubermaticv1.PresetSpec{
			Digitalocean: kubermaticv1.Digitalocean{},
			VSphere:      kubermaticv1.VSphere{},
			Openstack:    kubermaticv1.Openstack{},
			Hetzner:      kubermaticv1.Hetzner{},
			GCP:          kubermaticv1.GCP{},
			Azure:        kubermaticv1.Azure{},
			AWS:          kubermaticv1.AWS{},
			Packet:       kubermaticv1.Packet{},
			Fake:         kubermaticv1.Fake{},
			Kubevirt:     kubermaticv1.Kubevirt{},
		},
	}
}
