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

package openstack

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"
	osavailabilityzones "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	osflavors "github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	osprojects "github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	ostokens "github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	osusers "github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	osextnetwork "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	osrouters "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	ossecuritygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	ossecuritygrouprules "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	osnetworks "github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	osports "github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	ossubnets "github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"

	"k8c.io/kubermatic/v2/pkg/provider"
)

const (
	subnetCIDR         = "192.168.1.0/24"
	subnetFirstAddress = "192.168.1.2"
	subnetLastAddress  = "192.168.1.254"

	resourceNamePrefix = "kubernetes-"
)

func getSecurityGroups(netClient *gophercloud.ServiceClient, opts ossecuritygroups.ListOpts) ([]ossecuritygroups.SecGroup, error) {
	page, err := ossecuritygroups.List(netClient, opts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list security groups: %w", err)
	}
	secGroups, err := ossecuritygroups.ExtractGroups(page)
	if err != nil {
		return nil, fmt.Errorf("failed to extract security groups: %w", err)
	}
	return secGroups, nil
}

// NetworkWithExternalExt is a struct that implements all networks
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
	for _, n := range existingNetworks {
		if n.External == isExternal {
			candidates = append(candidates, &n)
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

func getExternalNetwork(netClient *gophercloud.ServiceClient) (*NetworkWithExternalExt, error) {
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

func validateSecurityGroupsExist(netClient *gophercloud.ServiceClient, securityGroups []string) error {
	for _, sg := range securityGroups {
		results, err := getSecurityGroups(netClient, ossecuritygroups.ListOpts{Name: sg})
		if err != nil {
			return fmt.Errorf("failed to get security group: %v", err)
		}
		if len(results) == 0 {
			return fmt.Errorf("specified security group %s not found", sg)
		}
	}
	return nil
}

func deleteSecurityGroup(netClient *gophercloud.ServiceClient, sgName string) error {
	results, err := getSecurityGroups(netClient, ossecuritygroups.ListOpts{Name: sgName})
	if err != nil {
		return fmt.Errorf("failed to get security group: %w", err)
	}

	for _, sg := range results {
		res := ossecuritygroups.Delete(netClient, sg.ID)
		if res.Err != nil {
			return res.Err
		}
		if err := res.ExtractErr(); err != nil {
			return err
		}
	}

	return nil
}

type createKubermaticSecurityGroupRequest struct {
	clusterName string
	lowPort     int
	highPort    int
}

func createKubermaticSecurityGroup(netClient *gophercloud.ServiceClient, req createKubermaticSecurityGroupRequest) (string, error) {
	secGroupName := resourceNamePrefix + req.clusterName
	secGroups, err := getSecurityGroups(netClient, ossecuritygroups.ListOpts{Name: secGroupName})
	if err != nil {
		return "", fmt.Errorf("failed to get security groups: %v", err)
	}

	var securityGroupID string
	switch len(secGroups) {
	case 0:
		gres := ossecuritygroups.Create(netClient, ossecuritygroups.CreateOpts{
			Name:        secGroupName,
			Description: "Contains security rules for the Kubernetes worker nodes",
		})
		if gres.Err != nil {
			return "", gres.Err
		}
		g, err := gres.Extract()
		if err != nil {
			return "", err
		}
		securityGroupID = g.ID
	case 1:
		securityGroupID = secGroups[0].ID
	default:
		return "", fmt.Errorf("there are already %d security groups with name %q, dont know which one to use",
			len(secGroups), secGroupName)
	}

	rules := []ossecuritygrouprules.CreateOpts{
		{
			// Allows ipv4 traffic within this group
			Direction:     ossecuritygrouprules.DirIngress,
			EtherType:     ossecuritygrouprules.EtherType4,
			SecGroupID:    securityGroupID,
			RemoteGroupID: securityGroupID,
		},
		{
			// Allows ipv6 traffic within this group
			Direction:     ossecuritygrouprules.DirIngress,
			EtherType:     ossecuritygrouprules.EtherType6,
			SecGroupID:    securityGroupID,
			RemoteGroupID: securityGroupID,
		},
		{
			// Allows ssh from external
			Direction:    ossecuritygrouprules.DirIngress,
			EtherType:    ossecuritygrouprules.EtherType4,
			SecGroupID:   securityGroupID,
			PortRangeMin: provider.DefaultSSHPort,
			PortRangeMax: provider.DefaultSSHPort,
			Protocol:     ossecuritygrouprules.ProtocolTCP,
		},
		{
			// Allows nodePorts from external
			Direction:    ossecuritygrouprules.DirIngress,
			EtherType:    ossecuritygrouprules.EtherType4,
			SecGroupID:   securityGroupID,
			PortRangeMin: req.lowPort,
			PortRangeMax: req.highPort,
			Protocol:     ossecuritygrouprules.ProtocolTCP,
		},
		{
			// Allows nodePorts from external
			Direction:    ossecuritygrouprules.DirIngress,
			EtherType:    ossecuritygrouprules.EtherType4,
			SecGroupID:   securityGroupID,
			PortRangeMin: req.lowPort,
			PortRangeMax: req.highPort,
			Protocol:     ossecuritygrouprules.ProtocolUDP,
		},
		{
			// Allows ICMP traffic
			Direction:  ossecuritygrouprules.DirIngress,
			EtherType:  ossecuritygrouprules.EtherType4,
			SecGroupID: securityGroupID,
			Protocol:   ossecuritygrouprules.ProtocolICMP,
		},
		{
			// Allows ICMPv6 traffic
			Direction:  ossecuritygrouprules.DirIngress,
			EtherType:  ossecuritygrouprules.EtherType6,
			SecGroupID: securityGroupID,
			Protocol:   ossecuritygrouprules.ProtocolIPv6ICMP,
		},
	}

	for _, opts := range rules {
	reiterate:
		rres := ossecuritygrouprules.Create(netClient, opts)
		if rres.Err != nil {
			if e, ok := rres.Err.(gophercloud.ErrUnexpectedResponseCode); ok && e.Actual == http.StatusConflict {
				// already exists
				continue
			}

			if _, ok := rres.Err.(gophercloud.ErrDefault400); ok && opts.Protocol == ossecuritygrouprules.ProtocolIPv6ICMP {
				// workaround for old versions of Opnestack with different protocol name,
				// from before https://review.opendev.org/#/c/252155/
				opts.Protocol = "icmpv6"
				goto reiterate // I'm very sorry, but this was really the cleanest way.
			}

			return "", rres.Err
		}

		if _, err := rres.Extract(); err != nil {
			return "", err
		}
	}

	return secGroupName, nil
}

func createKubermaticNetwork(netClient *gophercloud.ServiceClient, clusterName string) (*osnetworks.Network, error) {
	iTrue := true
	res := osnetworks.Create(netClient, osnetworks.CreateOpts{
		Name:         resourceNamePrefix + clusterName,
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

func deleteSubnet(netClient *gophercloud.ServiceClient, subnetID string) error {
	res := ossubnets.Delete(netClient, subnetID)
	if res.Err != nil {
		return res.Err
	}
	return res.ExtractErr()
}

func deleteRouter(netClient *gophercloud.ServiceClient, routerID string) error {
	res := osrouters.Delete(netClient, routerID)
	if res.Err != nil {
		return res.Err
	}
	return res.ExtractErr()
}

func createKubermaticSubnet(netClient *gophercloud.ServiceClient, clusterName, networkID string, dnsServers []string) (*ossubnets.Subnet, error) {
	iTrue := true
	res := ossubnets.Create(netClient, ossubnets.CreateOpts{
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
		DNSNameservers: dnsServers,
	})
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Extract()
}

func createKubermaticRouter(netClient *gophercloud.ServiceClient, clusterName, extNetworkName string) (*osrouters.Router, error) {
	extNetwork, err := getNetworkByName(netClient, extNetworkName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get external network %q: %v", extNetworkName, err)
	}

	iTrue := true
	gwi := osrouters.GatewayInfo{
		NetworkID: extNetwork.ID,
	}

	res := osrouters.Create(netClient, osrouters.CreateOpts{
		Name:         resourceNamePrefix + clusterName,
		AdminStateUp: &iTrue,
		GatewayInfo:  &gwi,
	})
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Extract()
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

func getFlavors(authClient *gophercloud.ProviderClient, region string) ([]osflavors.Flavor, error) {
	computeClient, err := goopenstack.NewComputeV2(authClient, gophercloud.EndpointOpts{Availability: gophercloud.AvailabilityPublic, Region: region})
	if err != nil {
		// this is special case for  services that span only one region.
		//nolint:gosimple
		//lint:ignore S1020 false positive, we must do the errcheck regardless of if its an ErrEndpointNotFound
		if _, ok := err.(*gophercloud.ErrEndpointNotFound); ok {
			computeClient, err = goopenstack.NewComputeV2(authClient, gophercloud.EndpointOpts{})
			if err != nil {
				return nil, fmt.Errorf("couldn't get identity endpoint: %v", err)
			}
		} else {
			return nil, fmt.Errorf("couldn't get identity endpoint: %v", err)
		}
	}

	var allFlavors []osflavors.Flavor
	pager := osflavors.ListDetail(computeClient, osflavors.ListOpts{})
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		flavors, err := osflavors.ExtractFlavors(page)
		if err != nil {
			return false, err
		}
		allFlavors = append(allFlavors, flavors...)
		return true, nil
	})

	if err != nil {
		return nil, err
	}
	return allFlavors, nil
}

func getTenants(authClient *gophercloud.ProviderClient, region string) ([]osprojects.Project, error) {
	sc, err := goopenstack.NewIdentityV3(authClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		// this is special case for  services that span only one region.
		//nolint:gosimple
		//lint:ignore S1020 false positive, we must do the errcheck regardless of if its an ErrEndpointNotFound
		if _, ok := err.(*gophercloud.ErrEndpointNotFound); ok {
			sc, err = goopenstack.NewIdentityV3(authClient, gophercloud.EndpointOpts{})
			if err != nil {
				return nil, fmt.Errorf("couldn't get identity endpoint: %v", err)
			}
		} else {
			return nil, fmt.Errorf("couldn't get identity endpoint: %v", err)
		}
	}

	// We need to fetch the token to get more details - here we're just fetching the user object from the token response
	user, err := ostokens.Get(sc, sc.Token()).ExtractUser()
	if err != nil {
		return nil, fmt.Errorf("couldn't get user from token: %v", err)
	}

	// We cannot list all projects - instead we must list projects of a given user
	allPages, err := osusers.ListProjects(sc, user.ID).AllPages()
	if err != nil {
		return nil, fmt.Errorf("couldn't list tenants: %v", err)
	}

	allProjects, err := osprojects.ExtractProjects(allPages)
	if err != nil {
		return nil, fmt.Errorf("couldn't extract tenants: %v", err)
	}

	return allProjects, nil
}

func getSubnetForNetwork(netClient *gophercloud.ServiceClient, networkIDOrName string) ([]ossubnets.Subnet, error) {
	var allSubnets []ossubnets.Subnet

	networks, err := getAllNetworks(netClient, osnetworks.ListOpts{Name: networkIDOrName})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %v", err)
	}

	networkID := networkIDOrName
	if len(networks) == 1 {
		networkID = networks[0].ID
	} else if len(networks) > 1 {
		return nil, fmt.Errorf("got %d networks for idOrName '%s', expected one at most", len(networks), networkIDOrName)
	}

	allPages, err := ossubnets.List(netClient, ossubnets.ListOpts{NetworkID: networkID}).AllPages()
	if err != nil {
		return nil, err
	}

	if allSubnets, err = ossubnets.ExtractSubnets(allPages); err != nil {
		return nil, err
	}

	return allSubnets, nil
}

func isNotFoundErr(err error) bool {
	var errNotFound gophercloud.ErrDefault404
	if errors.As(err, &errNotFound) || strings.Contains(err.Error(), "not found") {
		return true
	}
	return false
}

func getRouterIDForSubnet(netClient *gophercloud.ServiceClient, subnetID string) (string, error) {
	ports, err := getAllNetworkPorts(netClient, subnetID)
	if err != nil {
		return "", fmt.Errorf("failed to list ports for subnet: %v", err)
	}

	for _, port := range ports {
		if port.DeviceOwner == "network:router_interface" || port.DeviceOwner == "network:router_interface_distributed" {
			// Check IP for the interface & check if the IP belongs to the subnet
			return port.DeviceID, nil
		}
	}

	return "", nil
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

func getAvailabilityZones(computeClient *gophercloud.ServiceClient) ([]osavailabilityzones.AvailabilityZone, error) {
	allPages, err := osavailabilityzones.List(computeClient).AllPages()
	if err != nil {
		return nil, err
	}

	availabilityZones, err := osavailabilityzones.ExtractAvailabilityZones(allPages)
	if err != nil {
		return nil, err
	}

	return availabilityZones, nil
}
