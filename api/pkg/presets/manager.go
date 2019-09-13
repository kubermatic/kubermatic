package presets

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
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

// GetPresets returns presets which belong to the specific email group and for all users
func (m *Manager) GetPresets(userInfo provider.UserInfo) []kubermaticv1.Preset {
	presetList := make([]kubermaticv1.Preset, 0)

	for _, preset := range m.presets.Items {
		requiredEmailDomain := preset.Spec.RequiredEmailDomain
		// find preset for specific email domain
		if len(requiredEmailDomain) > 0 {
			userDomain := strings.Split(userInfo.Email, "@")
			if len(userDomain) == 2 && strings.EqualFold(userDomain[1], requiredEmailDomain) {
				presetList = append(presetList, preset)
			}
		} else {
			// find preset for "all" without RequiredEmailDomain field
			presetList = append(presetList, preset)
		}
	}

	return presetList
}

// GetPreset returns presets which belong to the specific email group and for all users
func (m *Manager) GetPreset(name string) *kubermaticv1.Preset {
	for _, preset := range m.presets.Items {
		if preset.Name == name {
			return &preset
		}
	}
	return emptyPreset()
}

func (m *Manager) SetCloudCredentials(presetName string, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error) {

	if cloud.VSphere != nil {
		return m.setVsphereCredentials(presetName, cloud)
	}
	if cloud.Openstack != nil {
		return m.setOpenStackCredentials(presetName, cloud, dc)
	}
	if cloud.Azure != nil {
		return m.setAzureCredentials(presetName, cloud)
	}
	if cloud.Digitalocean != nil {
		return m.setDigitalOceanCredentials(presetName, cloud)
	}
	if cloud.Packet != nil {
		return m.setPacketCredentials(presetName, cloud)
	}
	if cloud.Hetzner != nil {
		return m.setHetznerCredentials(presetName, cloud)
	}
	if cloud.AWS != nil {
		return m.setAWSCredentials(presetName, cloud)
	}
	if cloud.GCP != nil {
		return m.setGCPCredentials(presetName, cloud)
	}
	if cloud.Fake != nil {
		return m.setFakeCredentials(presetName, cloud)
	}
	if cloud.Kubevirt != nil {
		return m.setKubevirtCredentials(presetName, cloud)
	}

	return nil, fmt.Errorf("can not find provider to set credentials")
}

func emptyCredentialError(preset, provider string) error {
	return fmt.Errorf("the preset %s doesn't contain credential for %s provider", preset, provider)
}

func (m *Manager) setFakeCredentials(presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(presetName)

	if reflect.DeepEqual(preset.Spec.Fake.Credentials, kubermaticv1.FakePresetCredentials{}) {
		return nil, emptyCredentialError(presetName, "Fake")
	}

	cloud.Fake.Token = preset.Spec.Fake.Credentials.Token
	return &cloud, nil

}

func (m *Manager) setKubevirtCredentials(presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(presetName)
	if reflect.DeepEqual(preset.Spec.Kubevirt.Credentials, kubermaticv1.KubevirtPresetCredentials{}) {
		return nil, emptyCredentialError(presetName, "Kubevirt")
	}

	cloud.Kubevirt.Kubeconfig = preset.Spec.Kubevirt.Credentials.Kubeconfig
	return &cloud, nil
}

func (m *Manager) setGCPCredentials(presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(presetName)
	if reflect.DeepEqual(preset.Spec.GCP.Credentials, kubermaticv1.GCPPresetCredentials{}) {
		return nil, emptyCredentialError(presetName, "GCP")
	}

	credentials := preset.Spec.GCP.Credentials
	cloud.GCP.ServiceAccount = credentials.ServiceAccount
	cloud.GCP.Network = credentials.Network
	cloud.GCP.Subnetwork = credentials.Subnetwork
	return &cloud, nil

}

func (m *Manager) setAWSCredentials(presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(presetName)
	if reflect.DeepEqual(preset.Spec.AWS.Credentials, kubermaticv1.AWSPresetCredentials{}) {
		return nil, emptyCredentialError(presetName, "AWS")
	}

	credentials := preset.Spec.AWS.Credentials

	cloud.AWS.AccessKeyID = credentials.AccessKeyID
	cloud.AWS.SecretAccessKey = credentials.SecretAccessKey

	cloud.AWS.InstanceProfileName = credentials.InstanceProfileName
	cloud.AWS.RouteTableID = credentials.RouteTableID
	cloud.AWS.SecurityGroupID = credentials.SecurityGroupID
	cloud.AWS.VPCID = credentials.VPCID
	return &cloud, nil
}

func (m *Manager) setHetznerCredentials(presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(presetName)
	if reflect.DeepEqual(preset.Spec.Hetzner.Credentials, kubermaticv1.HetznerPresetCredentials{}) {
		return nil, emptyCredentialError(presetName, "Hetzner")
	}

	cloud.Hetzner.Token = preset.Spec.Hetzner.Credentials.Token
	return &cloud, nil

}

func (m *Manager) setPacketCredentials(presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(presetName)
	if reflect.DeepEqual(preset.Spec.Packet.Credentials, kubermaticv1.PacketPresetCredentials{}) {
		return nil, emptyCredentialError(presetName, "Packet")
	}

	credentials := preset.Spec.Packet.Credentials
	cloud.Packet.ProjectID = credentials.ProjectID
	cloud.Packet.APIKey = credentials.APIKey

	cloud.Packet.BillingCycle = credentials.BillingCycle
	if len(credentials.BillingCycle) == 0 {
		cloud.Packet.BillingCycle = "hourly"
	}

	return &cloud, nil

}

func (m *Manager) setDigitalOceanCredentials(presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(presetName)
	if reflect.DeepEqual(preset.Spec.Digitalocean.Credentials, kubermaticv1.DigitaloceanPresetCredentials{}) {
		return nil, emptyCredentialError(presetName, "Digitalocean")
	}

	cloud.Digitalocean.Token = preset.Spec.Digitalocean.Credentials.Token
	return &cloud, nil

}

func (m *Manager) setAzureCredentials(presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(presetName)
	if reflect.DeepEqual(preset.Spec.Azure.Credentials, kubermaticv1.AzurePresetCredentials{}) {
		return nil, emptyCredentialError(presetName, "Azure")
	}

	credentials := preset.Spec.Azure.Credentials
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

func (m *Manager) setOpenStackCredentials(presetName string, cloud kubermaticv1.CloudSpec, dc *kubermaticv1.Datacenter) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(presetName)
	if reflect.DeepEqual(preset.Spec.Openstack.Credentials, kubermaticv1.OpenstackPresetCredentials{}) {
		return nil, emptyCredentialError(presetName, "Openstack")
	}

	credentials := preset.Spec.Openstack.Credentials

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

func (m *Manager) setVsphereCredentials(presetName string, cloud kubermaticv1.CloudSpec) (*kubermaticv1.CloudSpec, error) {
	preset := m.GetPreset(presetName)
	if reflect.DeepEqual(preset.Spec.VSphere.Credentials, kubermaticv1.VSpherePresetCredentials{}) {
		return nil, emptyCredentialError(presetName, "Vsphere")
	}
	credentials := preset.Spec.VSphere.Credentials
	cloud.VSphere.Password = credentials.Password
	cloud.VSphere.Username = credentials.Username

	cloud.VSphere.VMNetName = credentials.VMNetName
	return &cloud, nil

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
