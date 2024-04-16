/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package openstack

import (
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud"
	osextnetwork "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	osnetworks "github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	osports "github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
)

// NetworkWithExternalExt is a struct that implements all networks.
type NetworkWithExternalExt struct {
	osnetworks.Network
	osextnetwork.NetworkExternalExt
}

func getAllNetworks(netClient *gophercloud.ServiceClient, opts osnetworks.ListOpts) ([]NetworkWithExternalExt, error) {
	var allNetworks []NetworkWithExternalExt
	allPages, err := osnetworks.List(netClient, opts).AllPages()
	if err != nil {
		return nil, err
	}

	if err = osnetworks.ExtractNetworksInto(allPages, &allNetworks); err != nil {
		return nil, err
	}

	return allNetworks, nil
}

func getNetworkByName(netClient *gophercloud.ServiceClient, name string, isExternal bool) (*NetworkWithExternalExt, error) {
	existingNetworks, err := getAllNetworks(netClient, osnetworks.ListOpts{Name: name})
	if err != nil {
		return nil, err
	}

	candidates := []*NetworkWithExternalExt{}
	for i, n := range existingNetworks {
		if n.External == isExternal {
			candidates = append(candidates, &existingNetworks[i])
		}
	}

	switch len(candidates) {
	case 1:
		return candidates[0], nil
	case 0:
		return nil, fmt.Errorf("network named '%s' with external=%v not found", name, isExternal)
	default:
		return nil, fmt.Errorf("found %d networks for name '%s' (external=%v), expected exactly one", len(candidates), name, isExternal)
	}
}

func getDefaultExternalNetwork(netClient *gophercloud.ServiceClient) (*NetworkWithExternalExt, error) {
	existingNetworks, err := getAllNetworks(netClient, osnetworks.ListOpts{})
	if err != nil {
		return nil, err
	}

	for _, n := range existingNetworks {
		if n.External {
			return &n, nil
		}
	}

	return nil, errors.New("no external network found")
}

func createUserClusterNetwork(netClient *gophercloud.ServiceClient, networkName string) (*osnetworks.Network, error) {
	iTrue := true
	res := osnetworks.Create(netClient, osnetworks.CreateOpts{
		Name:         networkName,
		AdminStateUp: &iTrue,
	})
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Extract()
}

func deleteNetworkByName(netClient *gophercloud.ServiceClient, networkName string) error {
	network, err := getNetworkByName(netClient, networkName, false)
	if err != nil {
		return fmt.Errorf("failed to get network '%s' by name: %w", networkName, err)
	}

	res := osnetworks.Delete(netClient, network.ID)
	if res.Err != nil {
		return res.Err
	}
	return res.ExtractErr()
}

func getAllNetworkPorts(netClient *gophercloud.ServiceClient, subnetID string) ([]osports.Port, error) {
	allPages, err := osports.List(netClient, osports.ListOpts{
		FixedIPs: []osports.FixedIPOpts{{SubnetID: subnetID}},
	}).AllPages()
	if err != nil {
		return nil, err
	}

	allPorts, err := osports.ExtractPorts(allPages)
	if err != nil {
		return nil, err
	}

	return allPorts, nil
}
