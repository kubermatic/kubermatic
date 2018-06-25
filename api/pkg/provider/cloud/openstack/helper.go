package openstack

import (
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"
	osflavors "github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	osprojects "github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	osextnetwork "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	osrouters "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	ossecuritygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	osecruritygrouprules "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	osnetworks "github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
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

type networkWithExternalExt struct {
	osnetworks.Network
	osextnetwork.NetworkExternalExt
}

func getAllNetworks(netClient *gophercloud.ServiceClient) ([]networkWithExternalExt, error) {
	var allNetworks []networkWithExternalExt
	allPages, err := osnetworks.List(netClient, nil).AllPages()
	if err != nil {
		return nil, err
	}

	if err = osnetworks.ExtractNetworksInto(allPages, &allNetworks); err != nil {
		return nil, err
	}

	return allNetworks, nil
}

func getNetworkByName(netClient *gophercloud.ServiceClient, network string, isExternal bool) (*networkWithExternalExt, error) {
	existingNetworks, err := getAllNetworks(netClient)
	if err != nil {
		return nil, err
	}

	for _, n := range existingNetworks {
		if n.Name == network && n.External == isExternal {
			return &n, nil
		}
	}

	return nil, errNotFound
}

func getExternalNetwork(netClient *gophercloud.ServiceClient) (*networkWithExternalExt, error) {
	existingNetworks, err := getAllNetworks(netClient)
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
		return err
	}

	res := osnetworks.Delete(netClient, network.ID)
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

	allPages, err := osprojects.List(sc, osprojects.ListOpts{}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("couldn't list tenants: %v", err)
	}

	allProjects, err := osprojects.ExtractProjects(allPages)
	if err != nil {
		return nil, err
	}

	return allProjects, nil
}
