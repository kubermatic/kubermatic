/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest/to"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

func deleteSubnet(ctx context.Context, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	subnetsClient, err := getSubnetsClient(cloud, credentials)
	if err != nil {
		return err
	}

	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}
	deleteSubnetFuture, err := subnetsClient.Delete(ctx, resourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName)
	if err != nil {
		return err
	}

	if err = deleteSubnetFuture.WaitForCompletionRef(ctx, subnetsClient.Client); err != nil {
		return err
	}

	return nil
}

func deleteAvailabilitySet(ctx context.Context, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	asClient, err := getAvailabilitySetClient(cloud, credentials)
	if err != nil {
		return err
	}

	_, err = asClient.Delete(ctx, cloud.Azure.ResourceGroup, cloud.Azure.AvailabilitySet)
	return err
}

// ensureResourceGroup will create or update an Azure resource group. The call is idempotent.
func ensureResourceGroup(ctx context.Context, cloud kubermaticv1.CloudSpec, location string, clusterName string, credentials Credentials) error {
	groupsClient, err := getGroupsClient(cloud, credentials)
	if err != nil {
		return err
	}

	parameters := resources.Group{
		Name:     to.StringPtr(cloud.Azure.ResourceGroup),
		Location: to.StringPtr(location),
		Tags: map[string]*string{
			clusterTagKey: to.StringPtr(clusterName),
		},
	}
	if _, err = groupsClient.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, parameters); err != nil {
		return fmt.Errorf("failed to create or update resource group %q: %v", cloud.Azure.ResourceGroup, err)
	}

	return nil
}

// ensureSecurityGroup will create or update an Azure security group. The call is idempotent.
func (a *Azure) ensureSecurityGroup(cloud kubermaticv1.CloudSpec, location string, clusterName string, portRangeLow int, portRangeHigh int, credentials Credentials) error {
	sgClient, err := getSecurityGroupsClient(cloud, credentials)
	if err != nil {
		return err
	}

	parameters := network.SecurityGroup{
		Name:     to.StringPtr(cloud.Azure.SecurityGroup),
		Location: to.StringPtr(location),
		Tags: map[string]*string{
			clusterTagKey: to.StringPtr(clusterName),
		},
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			Subnets: &[]network.Subnet{
				{
					Name: to.StringPtr(cloud.Azure.SubnetName),
					ID:   to.StringPtr(assembleSubnetID(cloud)),
				},
			},
			// inbound
			SecurityRules: &[]network.SecurityRule{
				{
					Name: to.StringPtr("ssh_ingress"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionInbound,
						Protocol:                 network.SecurityRuleProtocolTCP,
						SourceAddressPrefix:      to.StringPtr("*"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("22"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(100),
					},
				},
				{
					Name: to.StringPtr("inter_node_comm"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionInbound,
						Protocol:                 "*",
						SourceAddressPrefix:      to.StringPtr("VirtualNetwork"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("VirtualNetwork"),
						DestinationPortRange:     to.StringPtr("*"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(200),
					},
				},
				{
					Name: to.StringPtr("azure_load_balancer"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionInbound,
						Protocol:                 "*",
						SourceAddressPrefix:      to.StringPtr("AzureLoadBalancer"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("*"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(300),
					},
				},
				{
					// Allow access to node ports from everywhere
					Name: to.StringPtr("node_ports_ingress"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionInbound,
						Protocol:                 network.SecurityRuleProtocolAsterisk,
						SourceAddressPrefix:      to.StringPtr("*"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr(fmt.Sprintf("%d-%d", portRangeLow, portRangeHigh)),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(400),
					},
				},
				// outbound
				{
					Name: to.StringPtr("outbound_allow_all"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionOutbound,
						Protocol:                 "*",
						SourceAddressPrefix:      to.StringPtr("*"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("*"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(100),
					},
				},
			},
		},
	}

	updatedRules := append(*parameters.SecurityRules, tcpDenyAllRule(), udpDenyAllRule(), icmpAllowAllRule())
	parameters.SecurityRules = &updatedRules

	if _, err = sgClient.CreateOrUpdate(a.ctx, cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup, parameters); err != nil {
		return fmt.Errorf("failed to create or update resource group %q: %v", cloud.Azure.ResourceGroup, err)
	}

	return nil
}

// ensureVNet will create or update an Azure virtual network in the specified resource group. The call is idempotent.
func ensureVNet(ctx context.Context, cloud kubermaticv1.CloudSpec, location string, clusterName string, credentials Credentials) error {
	networksClient, err := getNetworksClient(cloud, credentials)
	if err != nil {
		return err
	}

	parameters := network.VirtualNetwork{
		Name:     to.StringPtr(cloud.Azure.VNetName),
		Location: to.StringPtr(location),
		Tags: map[string]*string{
			clusterTagKey: to.StringPtr(clusterName),
		},
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{AddressPrefixes: &[]string{"10.0.0.0/16"}},
		},
	}

	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}
	future, err := networksClient.CreateOrUpdate(ctx, resourceGroup, cloud.Azure.VNetName, parameters)
	if err != nil {
		return fmt.Errorf("failed to create or update virtual network %q: %v", cloud.Azure.VNetName, err)
	}

	if err = future.WaitForCompletionRef(ctx, networksClient.Client); err != nil {
		return fmt.Errorf("failed to create or update virtual network %q: %v", cloud.Azure.VNetName, err)
	}

	return nil
}

// ensureSubnet will create or update an Azure subnetwork in the specified vnet. The call is idempotent.
func ensureSubnet(ctx context.Context, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	subnetsClient, err := getSubnetsClient(cloud, credentials)
	if err != nil {
		return err
	}

	parameters := network.Subnet{
		Name: to.StringPtr(cloud.Azure.SubnetName),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr("10.0.0.0/16"),
		},
	}

	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}
	future, err := subnetsClient.CreateOrUpdate(ctx, resourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName, parameters)
	if err != nil {
		return fmt.Errorf("failed to create or update subnetwork %q: %v", cloud.Azure.SubnetName, err)
	}

	if err = future.WaitForCompletionRef(ctx, subnetsClient.Client); err != nil {
		return fmt.Errorf("failed to create or update subnetwork %q: %v", cloud.Azure.SubnetName, err)
	}

	return nil
}

// ensureRouteTable will create or update an Azure route table attached to the specified subnet. The call is idempotent.
func ensureRouteTable(ctx context.Context, cloud kubermaticv1.CloudSpec, location string, credentials Credentials) error {
	routeTablesClient, err := getRouteTablesClient(cloud, credentials)
	if err != nil {
		return err
	}

	parameters := network.RouteTable{
		Name:     to.StringPtr(cloud.Azure.RouteTableName),
		Location: to.StringPtr(location),
		RouteTablePropertiesFormat: &network.RouteTablePropertiesFormat{
			Subnets: &[]network.Subnet{
				{
					Name: to.StringPtr(cloud.Azure.SubnetName),
					ID:   to.StringPtr(assembleSubnetID(cloud)),
				},
			},
		},
	}

	future, err := routeTablesClient.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName, parameters)
	if err != nil {
		return fmt.Errorf("failed to create or update route table %q: %v", cloud.Azure.RouteTableName, err)
	}

	if err = future.WaitForCompletionRef(ctx, routeTablesClient.Client); err != nil {
		return fmt.Errorf("failed to create or update route table %q: %v", cloud.Azure.RouteTableName, err)
	}

	return nil
}

func ensureAvailabilitySet(ctx context.Context, name, location string, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	client, err := getAvailabilitySetClient(cloud, credentials)
	if err != nil {
		return err
	}

	faultDomainCount, ok := faultDomainsPerRegion[location]
	if !ok {
		return fmt.Errorf("could not determine the number of fault domains, unknown region %q", location)
	}

	as := compute.AvailabilitySet{
		Name:     to.StringPtr(name),
		Location: to.StringPtr(location),
		Sku: &compute.Sku{
			Name: to.StringPtr("Aligned"),
		},
		AvailabilitySetProperties: &compute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  to.Int32Ptr(faultDomainCount),
			PlatformUpdateDomainCount: to.Int32Ptr(20),
		},
	}

	_, err = client.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, name, as)
	return err
}

func deleteVNet(ctx context.Context, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	networksClient, err := getNetworksClient(cloud, credentials)
	if err != nil {
		return err
	}

	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}
	deleteVNetFuture, err := networksClient.Delete(ctx, resourceGroup, cloud.Azure.VNetName)
	if err != nil {
		return err
	}

	if err = deleteVNetFuture.WaitForCompletionRef(ctx, networksClient.Client); err != nil {
		return err
	}

	return nil
}

func deleteResourceGroup(ctx context.Context, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	groupsClient, err := getGroupsClient(cloud, credentials)
	if err != nil {
		return err
	}

	// We're doing a Get to see if its already gone or not.
	// We could also directly call delete but the error response would need to be unpacked twice to get the correct error message.
	// Doing a get is simpler.
	if _, err := groupsClient.Get(ctx, cloud.Azure.ResourceGroup); err != nil {
		return err
	}

	future, err := groupsClient.Delete(ctx, cloud.Azure.ResourceGroup)
	if err != nil {
		return err
	}

	if err = future.WaitForCompletionRef(ctx, groupsClient.Client); err != nil {
		return err
	}

	return nil
}

func deleteRouteTable(ctx context.Context, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	routeTablesClient, err := getRouteTablesClient(cloud, credentials)
	if err != nil {
		return err
	}

	future, err := routeTablesClient.Delete(ctx, cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName)
	if err != nil {
		return err
	}

	if err = future.WaitForCompletionRef(ctx, routeTablesClient.Client); err != nil {
		return err
	}

	return nil
}

func deleteSecurityGroup(ctx context.Context, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	securityGroupsClient, err := getSecurityGroupsClient(cloud, credentials)
	if err != nil {
		return err
	}

	future, err := securityGroupsClient.Delete(ctx, cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup)
	if err != nil {
		return err
	}

	if err = future.WaitForCompletionRef(ctx, securityGroupsClient.Client); err != nil {
		return err
	}

	return nil
}

func tcpDenyAllRule() network.SecurityRule {
	return network.SecurityRule{
		Name: to.StringPtr(denyAllTCPSecGroupRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Direction:                network.SecurityRuleDirectionInbound,
			Protocol:                 "TCP",
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
			DestinationPortRange:     to.StringPtr("*"),
			DestinationAddressPrefix: to.StringPtr("*"),
			Access:                   network.SecurityRuleAccessDeny,
			Priority:                 to.Int32Ptr(800),
		},
	}
}

func udpDenyAllRule() network.SecurityRule {
	return network.SecurityRule{
		Name: to.StringPtr(denyAllUDPSecGroupRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Direction:                network.SecurityRuleDirectionInbound,
			Protocol:                 "UDP",
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
			DestinationPortRange:     to.StringPtr("*"),
			DestinationAddressPrefix: to.StringPtr("*"),
			Access:                   network.SecurityRuleAccessDeny,
			Priority:                 to.Int32Ptr(801),
		},
	}
}

// Alright, so here's the deal. We need to allow ICMP, but on Azure it is not possible
// to specify ICMP as a protocol in a rule - only TCP or UDP.
// Therefore we're hacking around it by first blocking all incoming TCP and UDP
// and if these don't match, we have an "allow all" rule. Dirty, but the only way.
// See also: https://tinyurl.com/azure-allow-icmp
func icmpAllowAllRule() network.SecurityRule {
	return network.SecurityRule{
		Name: to.StringPtr(allowAllICMPSecGroupRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Direction:                network.SecurityRuleDirectionInbound,
			Protocol:                 "*",
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
			DestinationAddressPrefix: to.StringPtr("*"),
			DestinationPortRange:     to.StringPtr("*"),
			Access:                   network.SecurityRuleAccessAllow,
			Priority:                 to.Int32Ptr(900),
		},
	}
}
