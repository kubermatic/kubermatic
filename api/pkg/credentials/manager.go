package credentials

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
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
