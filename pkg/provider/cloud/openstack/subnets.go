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
	"fmt"

	"github.com/gophercloud/gophercloud"
	osrouters "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/subnetpools"
	ossubnets "github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"

	"k8s.io/utils/net"
	"k8s.io/utils/ptr"
)

func createSubnet(netClient *gophercloud.ServiceClient, clusterName, networkID string, dnsServers []string) (*ossubnets.Subnet, error) {
	iTrue := true
	subnetOpts := ossubnets.CreateOpts{
		Name:       resourceNamePrefix + clusterName,
		NetworkID:  networkID,
		IPVersion:  gophercloud.IPv4,
		CIDR:       subnetCIDR,
		GatewayIP:  nil,
		EnableDHCP: &iTrue,
		AllocationPools: []ossubnets.AllocationPool{
			{
				Start: subnetFirstAddress,
				End:   subnetLastAddress,
			},
		},
	}

	for _, s := range dnsServers {
		if net.IsIPv4String(s) {
			subnetOpts.DNSNameservers = append(subnetOpts.DNSNameservers, s)
		}
	}

	res := ossubnets.Create(netClient, subnetOpts)
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Extract()
}

func createIPv6Subnet(netClient *gophercloud.ServiceClient, clusterName, networkID, subnetPoolName string, dnsServers []string) (*ossubnets.Subnet, error) {
	subnetOpts := ossubnets.CreateOpts{
		Name:            resourceNamePrefix + clusterName + "-ipv6",
		NetworkID:       networkID,
		IPVersion:       gophercloud.IPv6,
		GatewayIP:       nil,
		EnableDHCP:      ptr.To(true),
		IPv6AddressMode: "dhcpv6-stateless",
		IPv6RAMode:      "dhcpv6-stateless",
	}
	subnetPoolID := ""

	// if IPv6 subnet pool name is provided - resolve to ID
	if subnetPoolName != "" {
		subnetPool, err := getSubnetPoolByName(netClient, subnetPoolName)
		if err != nil {
			return nil, err
		}
		subnetPoolID = subnetPool.ID
	}

	// if IPv6 subnet pool name is not provided - look for the default IPv6 subnet pool
	if subnetPoolID == "" {
		pools, err := getAllSubnetPools(netClient, subnetpools.ListOpts{IPVersion: 6, IsDefault: ptr.To(true)})
		if err != nil {
			return nil, err
		}
		if len(pools) > 0 {
			subnetPoolID = pools[0].ID
		}
	}

	if subnetPoolID != "" {
		subnetOpts.SubnetPoolID = subnetPoolID
	} else {
		// if no IPv6 subnet pool was provided / found, use the default IPv6 subnet CIDR
		subnetOpts.CIDR = defaultIPv6SubnetCIDR
	}

	for _, s := range dnsServers {
		if net.IsIPv6String(s) {
			subnetOpts.DNSNameservers = append(subnetOpts.DNSNameservers, s)
		}
	}

	res := ossubnets.Create(netClient, subnetOpts)
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Extract()
}

func getSubnetPoolByName(netClient *gophercloud.ServiceClient, name string) (*subnetpools.SubnetPool, error) {
	pools, err := getAllSubnetPools(netClient, subnetpools.ListOpts{Name: name})
	if err != nil {
		return nil, err
	}
	switch len(pools) {
	case 1:
		return &pools[0], nil
	case 0:
		return nil, fmt.Errorf("subnet pool named '%s' not found", name)
	default:
		return nil, fmt.Errorf("found %d subnet pools for name '%s', expected exactly one", len(pools), name)
	}
}

func getSubnetByID(netClient *gophercloud.ServiceClient, id string) (*ossubnets.Subnet, error) {
	listOpts := ossubnets.ListOpts{
		ID: id,
	}
	allSubnets, err := getAllSubnets(netClient, listOpts)
	if err != nil {
		return nil, err
	}
	switch len(allSubnets) {
	case 1:
		return &allSubnets[0], nil
	case 0:
		return nil, fmt.Errorf("subnet with id '%s' not found", id)
	default:
		return nil, fmt.Errorf("found %d subnets for id '%s', expected exactly one", len(allSubnets), id)
	}
}

func getSubnetByName(netClient *gophercloud.ServiceClient, name string) (*ossubnets.Subnet, error) {
	listOpts := ossubnets.ListOpts{
		Name: name,
	}
	allSubnets, err := getAllSubnets(netClient, listOpts)
	if err != nil {
		return nil, err
	}
	switch len(allSubnets) {
	case 1:
		return &allSubnets[0], nil
	case 0:
		return nil, fmt.Errorf("subnet named '%s' not found", name)
	default:
		return nil, fmt.Errorf("found %d subnets for name '%s', expected exactly one", len(allSubnets), name)
	}
}

func getAllSubnets(netClient *gophercloud.ServiceClient, listOpts ossubnets.ListOpts) ([]ossubnets.Subnet, error) {
	allPages, err := ossubnets.List(netClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}
	allSubnets, err := ossubnets.ExtractSubnets(allPages)
	if err != nil {
		return nil, err
	}
	return allSubnets, nil
}

func getAllSubnetPools(netClient *gophercloud.ServiceClient, listOpts subnetpools.ListOpts) ([]subnetpools.SubnetPool, error) {
	allPages, err := subnetpools.List(netClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}
	allSubnetPools, err := subnetpools.ExtractSubnetPools(allPages)
	if err != nil {
		return nil, err
	}
	return allSubnetPools, nil
}

func deleteSubnet(netClient *gophercloud.ServiceClient, subnetID string) error {
	res := ossubnets.Delete(netClient, subnetID)
	if res.Err != nil {
		return res.Err
	}
	return res.ExtractErr()
}

func attachSubnetToRouter(netClient *gophercloud.ServiceClient, subnetID, routerID string) (*osrouters.InterfaceInfo, error) {
	interf, err := func() (*osrouters.InterfaceInfo, error) {
		res := osrouters.AddInterface(netClient, routerID, osrouters.AddInterfaceOpts{
			SubnetID: subnetID,
		})
		if res.Err != nil {
			return nil, res.Err
		}
		return res.Extract()
	}()
	return interf, ignoreRouterAlreadyHasPortInSubnetError(err, subnetID)
}

func detachSubnetFromRouter(netClient *gophercloud.ServiceClient, subnetID, routerID string) (*osrouters.InterfaceInfo, error) {
	res := osrouters.RemoveInterface(netClient, routerID, osrouters.RemoveInterfaceOpts{
		SubnetID: subnetID,
	})
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Extract()
}
