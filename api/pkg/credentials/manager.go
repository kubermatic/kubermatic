package credentials

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"

	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

// Credentials specifies custom credentials for supported provider
type Credentials struct {
	Digitalocean []DigitaloceanCredentials `json:"digitalocean,omitempty"`
	Hetzner      []HetznerCredentials      `json:"hetzner,omitempty"`
	Azure        []AzureCredentials        `json:"azure,omitempty"`
	VSphere      []VSphereCredentials      `json:"vsphere,omitempty"`
	AWS          []AWSCredentials          `json:"aws,omitempty"`
	Openstack    []OpenstackCredentials    `json:"openstack,omitempty"`
	Packet       []PacketCredentials       `json:"packet,omitempty"`
	GCP          []GCPCredentials          `json:"gcp,omitempty"`
	Fake         []FakeCredentials         `json:"fake,omitempty"`
}

// DigitaloceanCredential defines Digitalocean credential
type DigitaloceanCredentials struct {
	Name  string `json:"name"`
	Token string `json:"token"` // Token is used to authenticate with the DigitalOcean API.
}

type HetznerCredentials struct {
	Name  string `json:"name"`
	Token string `json:"token"` // Token is used to authenticate with the Hetzner API.
}

type AzureCredentials struct {
	Name           string `json:"name"`
	TenantID       string `json:"tenantId"`
	SubscriptionID string `json:"subscriptionId"`
	ClientID       string `json:"clientId"`
	ClientSecret   string `json:"clientSecret"`
}

// VSphereCredentials credentials represents a credential for accessing vSphere
type VSphereCredentials struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type AWSCredentials struct {
	Name            string `json:"name"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
}

// OpenstackCredentials specifies access data to an openstack cloud.
type OpenstackCredentials struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	Tenant   string `json:"tenant"`
	Domain   string `json:"domain"`
}

// PacketCredentials specifies access data to a Packet cloud.
type PacketCredentials struct {
	Name      string `json:"name"`
	APIKey    string `json:"apiKey"`
	ProjectID string `json:"projectId"`
}

// GCPCredentials specifies access data to GCP.
type GCPCredentials struct {
	Name           string `json:"name"`
	ServiceAccount string `json:"serviceAccount"`
}

// FakeCredentials defines fake credential for tests
type FakeCredentials struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

// loadCredentials loads the custom credentials for supported providers
func loadCredentials(path string) (*Credentials, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		return nil, err
	}

	s := struct {
		Credentials *Credentials `json:"credentials"`
	}{}

	err = yaml.Unmarshal(bytes, &s)
	if err != nil {
		return nil, err
	}

	return s.Credentials, nil
}

// Manager is a object to handle credentials from a predefined config
type Manager struct {
	credentials *Credentials
}

func New() *Manager {
	credentials := &Credentials{}
	return &Manager{credentials: credentials}
}

// NewFromFiles returns a instance of manager with the credentials loaded from the given paths
func NewFromFiles(credentialsFilename string) (*Manager, error) {
	var credentials *Credentials
	var err error
	if len(credentialsFilename) > 0 {
		credentials, err = loadCredentials(credentialsFilename)
		if err != nil {
			return nil, fmt.Errorf("failed to load credentials from %s: %v", credentialsFilename, err)
		}
	}
	if credentials == nil {
		credentials = &Credentials{}
	}
	return &Manager{credentials: credentials}, nil
}

func (m *Manager) GetCredentials() *Credentials {
	return m.credentials
}

func (m *Manager) SetCloudCredentials(credentialName string, cloud v1.CloudSpec) (*v1.CloudSpec, error) {

	if cloud.VSphere != nil {
		return m.setVsphereCredentials(credentialName, cloud)
	}
	if cloud.Openstack != nil {
		return m.setOpenStackCredentials(credentialName, cloud)
	}
	if cloud.Azure != nil {
		return m.setAzureCredentials(credentialName, cloud)
	}
	if cloud.Digitalocean != nil {
		return m.setDigitalOceanCredentials(credentialName, cloud)
	}
	if cloud.Packet != nil {
		return m.setPacketCredentials(credentialName, cloud)
	}
	if cloud.Hetzner != nil {
		return m.setHetznerCredentials(credentialName, cloud)
	}
	if cloud.AWS != nil {
		return m.setAWSCredentials(credentialName, cloud)
	}
	if cloud.GCP != nil {
		return m.setGCPCredentials(credentialName, cloud)
	}
	if cloud.Fake != nil {
		return m.setFakeCredentials(credentialName, cloud)
	}

	return nil, fmt.Errorf("can not find provider to set credentials")
}

func emptyCredentialListError(provider string) error {
	return fmt.Errorf("can not find any credential for %s provider", provider)
}

func noCredentialError(credentialName string) error {
	return fmt.Errorf("can not find %s credential", credentialName)
}

func (m *Manager) setFakeCredentials(credentialName string, cloud v1.CloudSpec) (*v1.CloudSpec, error) {
	if m.credentials.Fake == nil {
		return nil, emptyCredentialListError("Fake")
	}
	for _, credential := range m.credentials.Fake {
		if credentialName == credential.Name {
			cloud.Fake.Token = credential.Token
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setGCPCredentials(credentialName string, cloud v1.CloudSpec) (*v1.CloudSpec, error) {
	if m.credentials.GCP == nil {
		return nil, emptyCredentialListError("GCP")
	}
	for _, credential := range m.credentials.GCP {
		if credentialName == credential.Name {
			cloud.GCP.ServiceAccount = credential.ServiceAccount
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setAWSCredentials(credentialName string, cloud v1.CloudSpec) (*v1.CloudSpec, error) {
	if m.credentials.AWS == nil {
		return nil, emptyCredentialListError("AWS")
	}
	for _, credential := range m.credentials.AWS {
		if credentialName == credential.Name {
			cloud.AWS.AccessKeyID = credential.AccessKeyID
			cloud.AWS.SecretAccessKey = credential.SecretAccessKey
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setHetznerCredentials(credentialName string, cloud v1.CloudSpec) (*v1.CloudSpec, error) {
	if m.credentials.Hetzner == nil {
		return nil, emptyCredentialListError("Hetzner")
	}
	for _, credential := range m.credentials.Hetzner {
		if credentialName == credential.Name {
			cloud.Hetzner.Token = credential.Token
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setPacketCredentials(credentialName string, cloud v1.CloudSpec) (*v1.CloudSpec, error) {
	if m.credentials.Packet == nil {
		return nil, emptyCredentialListError("Packet")
	}
	for _, credential := range m.credentials.Packet {
		if credentialName == credential.Name {
			cloud.Packet.ProjectID = credential.ProjectID
			cloud.Packet.APIKey = credential.APIKey
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setDigitalOceanCredentials(credentialName string, cloud v1.CloudSpec) (*v1.CloudSpec, error) {
	if m.credentials.Digitalocean == nil {
		return nil, emptyCredentialListError("Digitalocean")
	}
	for _, credential := range m.credentials.Digitalocean {
		if credentialName == credential.Name {
			cloud.Digitalocean.Token = credential.Token
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setAzureCredentials(credentialName string, cloud v1.CloudSpec) (*v1.CloudSpec, error) {
	if m.credentials.Azure == nil {
		return nil, emptyCredentialListError("Azure")
	}
	for _, credential := range m.credentials.Azure {
		if credentialName == credential.Name {
			cloud.Azure.TenantID = credential.TenantID
			cloud.Azure.ClientSecret = credential.ClientSecret
			cloud.Azure.ClientID = credential.ClientID
			cloud.Azure.SubscriptionID = credential.SubscriptionID
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setOpenStackCredentials(credentialName string, cloud v1.CloudSpec) (*v1.CloudSpec, error) {
	if m.credentials.Openstack == nil {
		return nil, emptyCredentialListError("Openstack")
	}
	for _, credential := range m.credentials.Openstack {
		if credentialName == credential.Name {
			cloud.Openstack.Username = credential.Username
			cloud.Openstack.Password = credential.Password
			cloud.Openstack.Domain = credential.Domain
			cloud.Openstack.Tenant = credential.Tenant
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}

func (m *Manager) setVsphereCredentials(credentialName string, cloud v1.CloudSpec) (*v1.CloudSpec, error) {
	if m.credentials.VSphere == nil {
		return nil, emptyCredentialListError("Vsphere")
	}
	for _, credential := range m.credentials.VSphere {
		if credentialName == credential.Name {
			cloud.VSphere.Password = credential.Password
			cloud.VSphere.Username = credential.Username
			return &cloud, nil
		}
	}
	return nil, noCredentialError(credentialName)
}
