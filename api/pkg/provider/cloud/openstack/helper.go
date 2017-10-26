package openstack

import (
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"
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

func getAllSecurityGroups(client *gophercloud.ProviderClient) ([]ossecuritygroups.SecGroup, error) {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}
	allGroups := []ossecuritygroups.SecGroup{}

	pager := ossecuritygroups.List(netClient, ossecuritygroups.ListOpts{})
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
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

func getAllNetworks(client *gophercloud.ProviderClient) ([]networkWithExternalExt, error) {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

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

func getNetworkByName(client *gophercloud.ProviderClient, network string, isExternal bool) (*networkWithExternalExt, error) {
	existingNetworks, err := getAllNetworks(client)
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

func getExternalNetwork(client *gophercloud.ProviderClient) (*networkWithExternalExt, error) {
	existingNetworks, err := getAllNetworks(client)
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

func validateSecurityGroupsExist(client *gophercloud.ProviderClient, securityGroups []string) error {
	existingGroups, err := getAllSecurityGroups(client)
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

func deleteSecurityGroup(client *gophercloud.ProviderClient, sgName string) error {
	securityGroups, err := getAllSecurityGroups(client)
	if err != nil {
		return err
	}

	for _, sg := range securityGroups {
		if sg.Name == sgName {
			netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
			if err != nil {
				return err
			}
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

func createKubermaticSecurityGroup(client *gophercloud.ProviderClient, clusterName string) (*ossecuritygroups.SecGroup, error) {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

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

func createKubermaticNetwork(client *gophercloud.ProviderClient, clusterName string) (*osnetworks.Network, error) {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

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

func deleteNetworkByName(client *gophercloud.ProviderClient, networkName string) error {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return err
	}

	network, err := getNetworkByName(client, networkName, false)
	if err != nil {
		return err
	}

	res := osnetworks.Delete(netClient, network.ID)
	if res.Err != nil {
		return res.Err
	}
	return res.ExtractErr()
}

func deleteRouter(client *gophercloud.ProviderClient, routerID string) error {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return err
	}

	res := osrouters.Delete(netClient, routerID)
	if res.Err != nil {
		return res.Err
	}
	return res.ExtractErr()
}

func createKubermaticSubnet(client *gophercloud.ProviderClient, clusterName, networkID string, dnsServers []string) (*ossubnets.Subnet, error) {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

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

func createKubermaticRouter(client *gophercloud.ProviderClient, clusterName, extNetworkName string) (*osrouters.Router, error) {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	extNetwork, err := getNetworkByName(client, extNetworkName, true)
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

func attachSubnetToRouter(client *gophercloud.ProviderClient, subnetID, routerID string) (*osrouters.InterfaceInfo, error) {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	res := osrouters.AddInterface(netClient, routerID, osrouters.AddInterfaceOpts{
		SubnetID: subnetID,
	})
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Extract()
}

func detachSubnetFromRouter(client *gophercloud.ProviderClient, subnetID, routerID string) (*osrouters.InterfaceInfo, error) {
	netClient, err := goopenstack.NewNetworkV2(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	res := osrouters.RemoveInterface(netClient, routerID, osrouters.RemoveInterfaceOpts{
		SubnetID: subnetID,
	})
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Extract()
}
