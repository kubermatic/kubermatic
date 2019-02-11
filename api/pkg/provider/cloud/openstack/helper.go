package openstack

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"
	osflavors "github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	osprojects "github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	ostokens "github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	osusers "github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	osextnetwork "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	osrouters "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	ossecuritygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	osecruritygrouprules "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	osnetworks "github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	osports "github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	ossubnets "github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

const (
	subnetCIDR         = "192.168.1.0/24"
	subnetFirstAddress = "192.168.1.2"
	subnetLastAddress  = "192.168.1.254"

	kubermaticNamePrefix = "kubermatic-"
)

var (
	errNotFound = errors.New("not found")
)

func getAllSecurityGroups(netClient *gophercloud.ServiceClient) ([]ossecuritygroups.SecGroup, error) {
	var allGroups []ossecuritygroups.SecGroup

	pager := ossecuritygroups.List(netClient, ossecuritygroups.ListOpts{})
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		securityGroups, err := ossecuritygroups.ExtractGroups(page)
		if err != nil {
			return false, err
		}
		allGroups = append(allGroups, securityGroups...)
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return allGroups, nil
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
		return nil, errNotFound
	default:
		return nil, fmt.Errorf("found %d external networks for name '%s', expected one at most", len(candidates), name)
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

	return nil, errNotFound
}

func securityGroupExistInList(name string, list []ossecuritygroups.SecGroup) bool {
	for _, gs := range list {
		if gs.Name == name {
			return true
		}
	}
	return false
}

func validateSecurityGroupsExist(netClient *gophercloud.ServiceClient, securityGroups []string) error {
	existingGroups, err := getAllSecurityGroups(netClient)
	if err != nil {
		return err
	}

	for _, sg := range securityGroups {
		if !securityGroupExistInList(sg, existingGroups) {
			return fmt.Errorf("specified security group %s not found", sg)
		}
	}
	return nil
}

func deleteSecurityGroup(netClient *gophercloud.ServiceClient, sgName string) error {
	securityGroups, err := getAllSecurityGroups(netClient)
	if err != nil {
		return err
	}

	for _, sg := range securityGroups {
		if sg.Name == sgName {
			res := ossecuritygroups.Delete(netClient, sg.ID)
			if res.Err != nil {
				return res.Err
			}
			if err := res.ExtractErr(); err != nil {
				return err
			}
		}
	}
	return nil
}

func createKubermaticSecurityGroup(netClient *gophercloud.ServiceClient, clusterName string) (*ossecuritygroups.SecGroup, error) {
	gres := ossecuritygroups.Create(netClient, ossecuritygroups.CreateOpts{
		Name:        kubermaticNamePrefix + clusterName,
		Description: "Contains security rules for the kubermatic worker nodes",
	})
	if gres.Err != nil {
		return nil, gres.Err
	}
	g, err := gres.Extract()
	if err != nil {
		return nil, err
	}

	rules := []osecruritygrouprules.CreateOpts{
		{
			// Allows ipv4 traffic within this group
			Direction:     osecruritygrouprules.DirIngress,
			EtherType:     osecruritygrouprules.EtherType4,
			SecGroupID:    g.ID,
			RemoteGroupID: g.ID,
		},
		{
			// Allows ipv6 traffic within this group
			Direction:     osecruritygrouprules.DirIngress,
			EtherType:     osecruritygrouprules.EtherType6,
			SecGroupID:    g.ID,
			RemoteGroupID: g.ID,
		},
		{
			// Allows ssh from external
			Direction:    osecruritygrouprules.DirIngress,
			EtherType:    osecruritygrouprules.EtherType4,
			SecGroupID:   g.ID,
			PortRangeMin: provider.DefaultSSHPort,
			PortRangeMax: provider.DefaultSSHPort,
			Protocol:     osecruritygrouprules.ProtocolTCP,
		},
		{
			// Allows kubelet from external
			Direction:    osecruritygrouprules.DirIngress,
			EtherType:    osecruritygrouprules.EtherType4,
			SecGroupID:   g.ID,
			PortRangeMin: provider.DefaultKubeletPort,
			PortRangeMax: provider.DefaultKubeletPort,
			Protocol:     osecruritygrouprules.ProtocolTCP,
		},
	}

	for _, opts := range rules {
		rres := osecruritygrouprules.Create(netClient, opts)
		if rres.Err != nil {
			return nil, rres.Err
		}

		if _, err := rres.Extract(); err != nil {
			return nil, err
		}
	}

	return g, nil
}

func createKubermaticNetwork(netClient *gophercloud.ServiceClient, clusterName string) (*osnetworks.Network, error) {
	iTrue := true
	res := osnetworks.Create(netClient, osnetworks.CreateOpts{
		Name:         kubermaticNamePrefix + clusterName,
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
		return fmt.Errorf("failed to get network '%s' by name: %v", networkName, err)
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
		Name:       kubermaticNamePrefix + clusterName,
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
		Name:         kubermaticNamePrefix + clusterName,
		AdminStateUp: &iTrue,
		GatewayInfo:  &gwi,
	})
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Extract()
}

func attachSubnetToRouter(netClient *gophercloud.ServiceClient, subnetID, routerID string) (*osrouters.InterfaceInfo, error) {
	res := osrouters.AddInterface(netClient, routerID, osrouters.AddInterfaceOpts{
		SubnetID: subnetID,
	})
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Extract()
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
		return nil, err
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
		return nil, fmt.Errorf("couldn't get identity endpoint: %v", err)
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
	if _, ok := err.(gophercloud.ErrDefault404); ok || strings.Contains(err.Error(), "not found") {
		return true
	}
	return false
}

func getRouterIDForSubnet(netClient *gophercloud.ServiceClient, subnetID, networkID string) (string, error) {
	ports, err := getAllNetworkPorts(netClient, networkID)
	if err != nil {
		return "", fmt.Errorf("failed to list ports for subnet: %v", err)
	}

	for _, port := range ports {
		if port.DeviceOwner == "network:router_interface" {
			// Check IP for the interface & check if the IP belongs to the subnet
			for _, ip := range port.FixedIPs {
				if ip.SubnetID == subnetID {
					return port.DeviceID, nil
				}
			}
		}
	}

	return "", errNotFound
}

func getAllNetworkPorts(netClient *gophercloud.ServiceClient, networkID string) ([]osports.Port, error) {
	allPages, err := osports.List(netClient, osports.ListOpts{
		NetworkID: networkID,
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
