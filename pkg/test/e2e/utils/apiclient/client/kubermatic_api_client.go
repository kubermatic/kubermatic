// Code generated by go-swagger; DO NOT EDIT.

package client

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/addon"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/admin"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/alibaba"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/anexia"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/aws"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/azure"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/constrainttemplates"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/credentials"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/datacenter"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/digitalocean"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/gcp"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/hetzner"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/metric"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/openstack"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/operations"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/packet"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/preset"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/project"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/seed"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/serviceaccounts"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/settings"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/tokens"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/users"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/versions"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/vsphere"
)

// Default kubermatic API HTTP client.
var Default = NewHTTPClient(nil)

const (
	// DefaultHost is the default Host
	// found in Meta (info) section of spec file
	DefaultHost string = "dev.kubermatic.io"
	// DefaultBasePath is the default BasePath
	// found in Meta (info) section of spec file
	DefaultBasePath string = "/"
)

// DefaultSchemes are the default schemes found in Meta (info) section of spec file
var DefaultSchemes = []string{"https"}

// NewHTTPClient creates a new kubermatic API HTTP client.
func NewHTTPClient(formats strfmt.Registry) *KubermaticAPI {
	return NewHTTPClientWithConfig(formats, nil)
}

// NewHTTPClientWithConfig creates a new kubermatic API HTTP client,
// using a customizable transport config.
func NewHTTPClientWithConfig(formats strfmt.Registry, cfg *TransportConfig) *KubermaticAPI {
	// ensure nullable parameters have default
	if cfg == nil {
		cfg = DefaultTransportConfig()
	}

	// create transport and client
	transport := httptransport.New(cfg.Host, cfg.BasePath, cfg.Schemes)
	return New(transport, formats)
}

// New creates a new kubermatic API client
func New(transport runtime.ClientTransport, formats strfmt.Registry) *KubermaticAPI {
	// ensure nullable parameters have default
	if formats == nil {
		formats = strfmt.Default
	}

	cli := new(KubermaticAPI)
	cli.Transport = transport
	cli.Addon = addon.New(transport, formats)
	cli.Admin = admin.New(transport, formats)
	cli.Alibaba = alibaba.New(transport, formats)
	cli.Anexia = anexia.New(transport, formats)
	cli.Aws = aws.New(transport, formats)
	cli.Azure = azure.New(transport, formats)
	cli.Constrainttemplates = constrainttemplates.New(transport, formats)
	cli.Credentials = credentials.New(transport, formats)
	cli.Datacenter = datacenter.New(transport, formats)
	cli.Digitalocean = digitalocean.New(transport, formats)
	cli.Gcp = gcp.New(transport, formats)
	cli.Hetzner = hetzner.New(transport, formats)
	cli.Metric = metric.New(transport, formats)
	cli.Openstack = openstack.New(transport, formats)
	cli.Operations = operations.New(transport, formats)
	cli.Packet = packet.New(transport, formats)
	cli.Preset = preset.New(transport, formats)
	cli.Project = project.New(transport, formats)
	cli.Seed = seed.New(transport, formats)
	cli.Serviceaccounts = serviceaccounts.New(transport, formats)
	cli.Settings = settings.New(transport, formats)
	cli.Tokens = tokens.New(transport, formats)
	cli.Users = users.New(transport, formats)
	cli.Versions = versions.New(transport, formats)
	cli.Vsphere = vsphere.New(transport, formats)
	return cli
}

// DefaultTransportConfig creates a TransportConfig with the
// default settings taken from the meta section of the spec file.
func DefaultTransportConfig() *TransportConfig {
	return &TransportConfig{
		Host:     DefaultHost,
		BasePath: DefaultBasePath,
		Schemes:  DefaultSchemes,
	}
}

// TransportConfig contains the transport related info,
// found in the meta section of the spec file.
type TransportConfig struct {
	Host     string
	BasePath string
	Schemes  []string
}

// WithHost overrides the default host,
// provided by the meta section of the spec file.
func (cfg *TransportConfig) WithHost(host string) *TransportConfig {
	cfg.Host = host
	return cfg
}

// WithBasePath overrides the default basePath,
// provided by the meta section of the spec file.
func (cfg *TransportConfig) WithBasePath(basePath string) *TransportConfig {
	cfg.BasePath = basePath
	return cfg
}

// WithSchemes overrides the default schemes,
// provided by the meta section of the spec file.
func (cfg *TransportConfig) WithSchemes(schemes []string) *TransportConfig {
	cfg.Schemes = schemes
	return cfg
}

// KubermaticAPI is a client for kubermatic API
type KubermaticAPI struct {
	Addon addon.ClientService

	Admin admin.ClientService

	Alibaba alibaba.ClientService

	Anexia anexia.ClientService

	Aws aws.ClientService

	Azure azure.ClientService

	Constrainttemplates constrainttemplates.ClientService

	Credentials credentials.ClientService

	Datacenter datacenter.ClientService

	Digitalocean digitalocean.ClientService

	Gcp gcp.ClientService

	Hetzner hetzner.ClientService

	Metric metric.ClientService

	Openstack openstack.ClientService

	Operations operations.ClientService

	Packet packet.ClientService

	Preset preset.ClientService

	Project project.ClientService

	Seed seed.ClientService

	Serviceaccounts serviceaccounts.ClientService

	Settings settings.ClientService

	Tokens tokens.ClientService

	Users users.ClientService

	Versions versions.ClientService

	Vsphere vsphere.ClientService

	Transport runtime.ClientTransport
}

// SetTransport changes the transport on the client and all its subresources
func (c *KubermaticAPI) SetTransport(transport runtime.ClientTransport) {
	c.Transport = transport
	c.Addon.SetTransport(transport)
	c.Admin.SetTransport(transport)
	c.Alibaba.SetTransport(transport)
	c.Anexia.SetTransport(transport)
	c.Aws.SetTransport(transport)
	c.Azure.SetTransport(transport)
	c.Constrainttemplates.SetTransport(transport)
	c.Credentials.SetTransport(transport)
	c.Datacenter.SetTransport(transport)
	c.Digitalocean.SetTransport(transport)
	c.Gcp.SetTransport(transport)
	c.Hetzner.SetTransport(transport)
	c.Metric.SetTransport(transport)
	c.Openstack.SetTransport(transport)
	c.Operations.SetTransport(transport)
	c.Packet.SetTransport(transport)
	c.Preset.SetTransport(transport)
	c.Project.SetTransport(transport)
	c.Seed.SetTransport(transport)
	c.Serviceaccounts.SetTransport(transport)
	c.Settings.SetTransport(transport)
	c.Tokens.SetTransport(transport)
	c.Users.SetTransport(transport)
	c.Versions.SetTransport(transport)
	c.Vsphere.SetTransport(transport)
}
