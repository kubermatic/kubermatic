/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package vmwareclouddirector

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/vmware/go-vcloud-director/v2/govcd"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

type Auth struct {
	Username      string
	Password      string
	APIToken      string
	Organization  string
	URL           string
	VDC           string
	AllowInsecure bool
}

type Client struct {
	Auth      *Auth
	VCDClient *govcd.VCDClient
}

func NewClient(spec kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc, dc *kubermaticv1.DatacenterSpecVMwareCloudDirector) (*Client, error) {
	creds, err := GetCredentialsForCluster(spec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	client, err := NewClientWithCreds(creds.Username, creds.Password, creds.APIToken, creds.Organization, creds.VDC, dc.URL, dc.AllowInsecure)
	if err != nil {
		return nil, fmt.Errorf("failed to create VMware Cloud Director client: %w", err)
	}
	return client, err
}

func NewClientWithCreds(username, password, apiToken, org, vdc, url string, allowInsecure bool) (*Client, error) {
	client := Client{
		Auth: &Auth{
			Username:      username,
			Password:      password,
			APIToken:      apiToken,
			Organization:  org,
			URL:           url,
			AllowInsecure: allowInsecure,
			VDC:           vdc,
		},
	}

	vcdClient, err := client.GetAuthenticatedClient()
	if err != nil {
		return nil, err
	}

	client.VCDClient = vcdClient
	return &client, nil
}

func NewClientWithAuth(auth Auth) (*Client, error) {
	client := Client{
		Auth: &auth,
	}

	vcdClient, err := client.GetAuthenticatedClient()
	if err != nil {
		return nil, err
	}

	client.VCDClient = vcdClient
	return &client, nil
}

func GetAuthInfo(spec kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc, dc *kubermaticv1.DatacenterSpecVMwareCloudDirector) (*Auth, error) {
	creds, err := GetCredentialsForCluster(spec, secretKeySelector)
	if err != nil {
		return nil, err
	}

	return &Auth{
		Username:      creds.Username,
		Password:      creds.Password,
		APIToken:      creds.APIToken,
		Organization:  creds.Organization,
		URL:           dc.URL,
		AllowInsecure: dc.AllowInsecure,
		VDC:           creds.VDC,
	}, nil
}

func (c *Client) GetAuthenticatedClient() (*govcd.VCDClient, error) {
	// Ensure that all required fields for authentication are provided
	// Fail early, without any API calls, if some required field is missing.
	if c.Auth == nil {
		return nil, fmt.Errorf("authentication configuration not provided")
	}
	// If API token is provided, use it for authentication.
	if c.Auth.APIToken == "" {
		if c.Auth.Username == "" {
			return nil, fmt.Errorf("username not provided")
		}
		if c.Auth.Password == "" {
			return nil, fmt.Errorf("password not provided")
		}
	}
	if c.Auth.URL == "" {
		return nil, fmt.Errorf("URL not provided")
	}
	if c.Auth.Organization == "" {
		return nil, fmt.Errorf("organization name not provided")
	}
	if c.Auth.VDC == "" {
		return nil, fmt.Errorf("vdc not provided")
	}

	// Ensure that `/api` suffix exists in the cloud director URL.
	apiEndpoint, err := url.Parse(c.Auth.URL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse url '%s': %w", c.Auth.URL, err)
	}
	if !strings.HasSuffix(c.Auth.URL, "/api") {
		apiEndpoint.Path = path.Join(apiEndpoint.Path, "api")
	}

	vcdClient := govcd.NewVCDClient(*apiEndpoint, c.Auth.AllowInsecure)

	if c.Auth.APIToken != "" {
		err = vcdClient.SetToken(c.Auth.Organization, govcd.ApiTokenHeader, c.Auth.APIToken)
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate with VMware Cloud Director using API Token: %w", err)
		}
		return vcdClient, nil
	}

	err = vcdClient.Authenticate(c.Auth.Username, c.Auth.Password, c.Auth.Organization)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with VMware Cloud Director: %w", err)
	}

	return vcdClient, nil
}

func (c *Client) GetOrganization() (*govcd.Org, error) {
	if c.Auth.Organization == "" {
		return nil, errors.New("organization must be configured")
	}

	org, err := c.VCDClient.GetOrgByNameOrId(c.Auth.Organization)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization '%s': %w", c.Auth.Organization, err)
	}
	return org, err
}

func (c *Client) GetVDCForOrg(org govcd.Org) (*govcd.Vdc, error) {
	if c.Auth.VDC == "" {
		return nil, errors.New("Organization VDC must be configured")
	}
	vcd, err := org.GetVDCByNameOrId(c.Auth.VDC, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization VDC '%s': %w", c.Auth.VDC, err)
	}
	return vcd, err
}
