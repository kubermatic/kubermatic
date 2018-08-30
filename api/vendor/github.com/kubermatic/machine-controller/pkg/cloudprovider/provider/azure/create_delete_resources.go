package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/types"
)

// deleteInterfacesByMachineUID will remove all network interfaces tagged with the specific machine's UID.
// The machine has to be deleted or disassociated with the interfaces beforehand, since Azure won't allow
// us to remove interfaces connected to a VM.
func deleteInterfacesByMachineUID(ctx context.Context, c *config, machineUID types.UID) error {
	ifClient, err := getInterfacesClient(c)
	if err != nil {
		return fmt.Errorf("failed to create interfaces client: %v", err)
	}

	list, err := ifClient.List(ctx, c.ResourceGroup)
	if err != nil {
		return fmt.Errorf("failed to list interfaces in resource group %q", c.ResourceGroup)
	}

	var allInterfaces []network.Interface

	for list.NotDone() {
		allInterfaces = append(allInterfaces, list.Values()...)
		if err = list.Next(); err != nil {
			return fmt.Errorf("failed to iterate the result list: %s", err)
		}
	}

	for _, iface := range allInterfaces {
		if iface.Tags != nil && iface.Tags[machineUIDTag] != nil && *iface.Tags[machineUIDTag] == string(machineUID) {
			future, err := ifClient.Delete(ctx, c.ResourceGroup, *iface.Name)
			if err != nil {
				return err
			}

			if err = future.WaitForCompletion(ctx, ifClient.Client); err != nil {
				return err
			}
		}
	}

	return nil
}

// deleteIPAddressesByMachineUID will remove public IP addresses tagged with the specific machine's UID.
// Their respective network interfaces have to be deleted or disassociated with the IPs beforehand, since
// Azure won't allow us to remove IPs connected to NICs.
func deleteIPAddressesByMachineUID(ctx context.Context, c *config, machineUID types.UID) error {
	ipClient, err := getIPClient(c)
	if err != nil {
		return fmt.Errorf("failed to create IP addresses client: %v", err)
	}

	list, err := ipClient.List(ctx, c.ResourceGroup)
	if err != nil {
		return fmt.Errorf("failed to list public IP addresses in resource group %q", c.ResourceGroup)
	}

	var allIPs []network.PublicIPAddress

	for list.NotDone() {
		allIPs = append(allIPs, list.Values()...)
		if err = list.Next(); err != nil {
			return fmt.Errorf("failed to iterate the result list: %s", err)
		}
	}

	for _, ip := range allIPs {
		if ip.Tags != nil && ip.Tags[machineUIDTag] != nil && *ip.Tags[machineUIDTag] == string(machineUID) {
			future, err := ipClient.Delete(ctx, c.ResourceGroup, *ip.Name)
			if err != nil {
				return err
			}

			if err = future.WaitForCompletion(ctx, ipClient.Client); err != nil {
				return err
			}
		}
	}

	return nil
}

func deleteVMsByMachineUID(ctx context.Context, c *config, machineUID types.UID) error {
	vmClient, err := getVMClient(c)
	if err != nil {
		return err
	}

	list, err := vmClient.ListAll(ctx)
	if err != nil {
		return err
	}

	var allServers []compute.VirtualMachine

	for list.NotDone() {
		allServers = append(allServers, list.Values()...)
		if err = list.Next(); err != nil {
			return fmt.Errorf("failed to iterate the result list: %s", err)
		}
	}

	for _, vm := range allServers {
		if vm.Tags != nil && vm.Tags[machineUIDTag] != nil && *vm.Tags[machineUIDTag] == string(machineUID) {
			future, err := vmClient.Delete(ctx, c.ResourceGroup, *vm.Name)
			if err != nil {
				return err
			}

			if err = future.WaitForCompletion(ctx, vmClient.Client); err != nil {
				return err
			}
		}
	}

	return nil
}

func deleteDisksByMachineUID(ctx context.Context, c *config, machineUID types.UID) error {
	disksClient, err := getDisksClient(c)
	if err != nil {
		return err
	}

	list, err := disksClient.List(ctx)
	if err != nil {
		return err
	}

	var allDisks []compute.Disk

	for list.NotDone() {
		allDisks = append(allDisks, list.Values()...)
		if err = list.Next(); err != nil {
			return fmt.Errorf("failed to iterate the result list: %s", err)
		}
	}

	for _, disk := range allDisks {
		if disk.Tags != nil && disk.Tags[machineUIDTag] != nil && *disk.Tags[machineUIDTag] == string(machineUID) {
			future, err := disksClient.Delete(ctx, c.ResourceGroup, *disk.Name)
			if err != nil {
				return err
			}

			if err = future.WaitForCompletion(ctx, disksClient.Client); err != nil {
				return err
			}
		}
	}

	return nil
}

func createPublicIPAddress(ctx context.Context, ipName string, machineUID types.UID, c *config) (*network.PublicIPAddress, error) {
	glog.Infof("Creating public IP %q", ipName)
	ipClient, err := getIPClient(c)
	if err != nil {
		return nil, err
	}

	ipParams := network.PublicIPAddress{
		Name:     to.StringPtr(ipName),
		Location: to.StringPtr(c.Location),
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion:   network.IPv4,
			PublicIPAllocationMethod: network.Static,
		},
		Tags: map[string]*string{machineUIDTag: to.StringPtr(string(machineUID))},
	}
	future, err := ipClient.CreateOrUpdate(ctx, c.ResourceGroup, ipName, ipParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create public IP address: %v", err)
	}

	err = future.WaitForCompletion(ctx, ipClient.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve public IP address creation result: %v", err)
	}

	if _, err = future.Result(*ipClient); err != nil {
		return nil, fmt.Errorf("failed to create public IP address: %v", err)
	}

	glog.Infof("Fetching info for IP address %q", ipName)
	ip, err := getPublicIPAddress(ctx, ipName, c.ResourceGroup, ipClient)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch info about public IP %q: %v", ipName, err)
	}

	return ip, nil
}

func getPublicIPAddress(ctx context.Context, ipName string, resourceGroup string, ipClient *network.PublicIPAddressesClient) (*network.PublicIPAddress, error) {
	ip, err := ipClient.Get(ctx, resourceGroup, ipName, "")
	if err != nil {
		return nil, err
	}

	return &ip, nil
}

func getSubnet(ctx context.Context, c *config) (network.Subnet, error) {
	subnetsClient, err := getSubnetsClient(c)
	if err != nil {
		return network.Subnet{}, fmt.Errorf("failed to create subnets client: %v", err)
	}

	return subnetsClient.Get(ctx, c.ResourceGroup, c.VNetName, c.SubnetName, "")
}

func getVirtualNetwork(ctx context.Context, c *config) (network.VirtualNetwork, error) {
	virtualNetworksClient, err := getVirtualNetworksClient(c)
	if err != nil {
		return network.VirtualNetwork{}, err
	}

	return virtualNetworksClient.Get(ctx, c.ResourceGroup, c.VNetName, "")
}

func createNetworkInterface(ctx context.Context, ifName string, machineUID types.UID, config *config, publicIP *network.PublicIPAddress) (network.Interface, error) {
	ifClient, err := getInterfacesClient(config)
	if err != nil {
		return network.Interface{}, fmt.Errorf("failed to create interfaces client: %v", err)
	}

	subnet, err := getSubnet(ctx, config)
	if err != nil {
		return network.Interface{}, fmt.Errorf("failed to fetch subnet: %v", err)
	}

	ifSpec := network.Interface{
		Name:     to.StringPtr(ifName),
		Location: &config.Location,
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: to.StringPtr("ip-config-1"),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Subnet: &subnet,
						PrivateIPAllocationMethod: network.Dynamic,
						PublicIPAddress:           publicIP,
					},
				},
			},
		},
		Tags: map[string]*string{machineUIDTag: to.StringPtr(string(machineUID))},
	}
	glog.Infof("Creating public network interface %q", ifName)
	future, err := ifClient.CreateOrUpdate(ctx, config.ResourceGroup, ifName, ifSpec)
	if err != nil {
		return network.Interface{}, fmt.Errorf("failed to create interface: %v", err)
	}

	err = future.WaitForCompletion(ctx, ifClient.Client)
	if err != nil {
		return network.Interface{}, fmt.Errorf("failed to get interface creation response: %v", err)
	}

	_, err = future.Result(*ifClient)
	if err != nil {
		return network.Interface{}, fmt.Errorf("failed to get interface creation result: %v", err)
	}

	glog.Infof("Fetching info about network interface %q", ifName)
	iface, err := ifClient.Get(ctx, config.ResourceGroup, ifName, "")
	if err != nil {
		return network.Interface{}, fmt.Errorf("failed to fetch info about interface %q: %v", ifName, err)
	}

	return iface, nil
}
