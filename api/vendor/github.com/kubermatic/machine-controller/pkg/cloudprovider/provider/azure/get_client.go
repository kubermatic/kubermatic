/*
Copyright 2019 The Machine Controller Authors.

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

package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

func getIPClient(c *config) (*network.PublicIPAddressesClient, error) {
	var err error
	ipClient := network.NewPublicIPAddressesClient(c.SubscriptionID)
	ipClient.Authorizer, err = auth.NewClientCredentialsConfig(c.ClientID, c.ClientSecret, c.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &ipClient, nil
}

func getSubnetsClient(c *config) (*network.SubnetsClient, error) {
	var err error
	subnetClient := network.NewSubnetsClient(c.SubscriptionID)
	subnetClient.Authorizer, err = auth.NewClientCredentialsConfig(c.ClientID, c.ClientSecret, c.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &subnetClient, nil
}

func getVirtualNetworksClient(c *config) (*network.VirtualNetworksClient, error) {
	var err error
	virtualNetworksClient := network.NewVirtualNetworksClient(c.SubscriptionID)
	virtualNetworksClient.Authorizer, err = auth.NewClientCredentialsConfig(c.ClientID, c.ClientSecret, c.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %v", err)
	}
	return &virtualNetworksClient, nil
}

func getVMClient(c *config) (*compute.VirtualMachinesClient, error) {
	var err error
	vmClient := compute.NewVirtualMachinesClient(c.SubscriptionID)
	vmClient.Authorizer, err = auth.NewClientCredentialsConfig(c.ClientID, c.ClientSecret, c.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &vmClient, nil
}

func getInterfacesClient(c *config) (*network.InterfacesClient, error) {
	var err error
	ifClient := network.NewInterfacesClient(c.SubscriptionID)
	ifClient.Authorizer, err = auth.NewClientCredentialsConfig(c.ClientID, c.ClientSecret, c.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &ifClient, err
}

func getDisksClient(c *config) (*compute.DisksClient, error) {
	var err error
	disksClient := compute.NewDisksClient(c.SubscriptionID)
	disksClient.Authorizer, err = auth.NewClientCredentialsConfig(c.ClientID, c.ClientSecret, c.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &disksClient, err
}
