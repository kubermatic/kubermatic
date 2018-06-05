package azure

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	clusterTagKey = "cluster"
)

type azure struct {
	dcs map[string]provider.DatacenterMeta
}

// New returns a new Azure provider.
func New(datacenters map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &azure{
		dcs: datacenters,
	}
}

func deleteSubnet(cloud *kubermaticv1.CloudSpec) error {
	subnetsClient, err := getSubnetsClient(cloud)
	if err != nil {
		return err
	}

	deleteSubnetFuture, err := subnetsClient.Delete(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName)
	if err != nil {
		return fmt.Errorf("failed to delete sub-network %q: %v", cloud.Azure.SubnetName, err)
	}

	if err = deleteSubnetFuture.WaitForCompletion(context.TODO(), subnetsClient.Client); err != nil {
		return fmt.Errorf("failed to delete sub-network %q: %v", cloud.Azure.SubnetName, err)
	}

	return nil
}

func deleteVNet(cloud *kubermaticv1.CloudSpec) error {
	networksClient, err := getNetworksClient(cloud)
	if err != nil {
		return err
	}

	deleteVNetFuture, err := networksClient.Delete(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.VNetName)
	if err != nil {
		return fmt.Errorf("failed to delete virtual network %q: %v", cloud.Azure.VNetName, err)
	}

	if err = deleteVNetFuture.WaitForCompletion(context.TODO(), networksClient.Client); err != nil {
		return fmt.Errorf("failed to delete virtual network %q: %v", cloud.Azure.VNetName, err)
	}

	return nil
}

func deleteResourceGroup(cloud *kubermaticv1.CloudSpec) error {
	groupsClient, err := getGroupsClient(cloud)
	if err != nil {
		return err
	}

	future, err := groupsClient.Delete(context.TODO(), cloud.Azure.ResourceGroup)
	if err != nil {
		return fmt.Errorf("failed to delete resource group %q: %v", cloud.Azure.ResourceGroup, err)
	}

	if err = future.WaitForCompletion(context.TODO(), groupsClient.Client); err != nil {
		return fmt.Errorf("failed to delete resource group %q: %v", cloud.Azure.ResourceGroup, err)
	}

	return nil
}

func deleteRouteTable(cloud *kubermaticv1.CloudSpec) error {
	routeTablesClient, err := getRouteTablesClient(cloud)
	if err != nil {
		return err
	}

	future, err := routeTablesClient.Delete(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName)
	if err != nil {
		return fmt.Errorf("failed to delete route table %q: %v", cloud.Azure.RouteTableName, err)
	}

	if err = future.WaitForCompletion(context.TODO(), routeTablesClient.Client); err != nil {
		return fmt.Errorf("failed to delete route table %q: %v", cloud.Azure.RouteTableName, err)
	}

	return nil
}

func (a *azure) CleanUpCloudProvider(cloud *kubermaticv1.CloudSpec) error {
	// TODO delete security group

	if err := deleteRouteTable(cloud); err != nil {
		return err
	}

	if err := deleteSubnet(cloud); err != nil {
		return err
	}

	if err := deleteVNet(cloud); err != nil {
		return err
	}

	if err := deleteResourceGroup(cloud); err != nil {
		return err
	}

	return nil
}

// createResourceGroup will create or update an Azure resource group. The call is idempotent.
func createResourceGroup(cloud *kubermaticv1.CloudSpec, location string, clusterName string) error {
	groupsClient, err := getGroupsClient(cloud)
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
	if _, err = groupsClient.CreateOrUpdate(context.TODO(), cloud.Azure.ResourceGroup, parameters); err != nil {
		return fmt.Errorf("failed to create or update resource group %q: %v", cloud.Azure.ResourceGroup, err)
	}

	return nil
}

// createVNet will create or update an Azure virtual network in the specified resource group. The call is idempotent.
func createVNet(cloud *kubermaticv1.CloudSpec, clusterName string) error {
	networksClient, err := getNetworksClient(cloud)
	if err != nil {
		return err
	}

	parameters := network.VirtualNetwork{
		Name: to.StringPtr(cloud.Azure.VNetName),
		Tags: map[string]*string{
			clusterTagKey: to.StringPtr(clusterName),
		},
	}

	future, err := networksClient.CreateOrUpdate(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.VNetName, parameters)
	if err != nil {
		return fmt.Errorf("failed to create or update virtual network %q: %v", cloud.Azure.VNetName, err)
	}

	if err = future.WaitForCompletion(context.TODO(), networksClient.Client); err != nil {
		return fmt.Errorf("failed to create or update virtual network %q: %v", cloud.Azure.VNetName, err)
	}

	return nil
}

// createSubnet will create or update an Azure subnetwork in the specified vnet. The call is idempotent.
func createSubnet(cloud *kubermaticv1.CloudSpec) error {
	subnetsClient, err := getSubnetsClient(cloud)
	if err != nil {
		return err
	}

	parameters := network.Subnet{
		Name: to.StringPtr(cloud.Azure.SubnetName),
	}

	future, err := subnetsClient.CreateOrUpdate(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName, parameters)
	if err != nil {
		return fmt.Errorf("failed to create or update subnetwork %q: %v", cloud.Azure.SubnetName, err)
	}

	if err = future.WaitForCompletion(context.TODO(), subnetsClient.Client); err != nil {
		return fmt.Errorf("failed to create or update subnetwork %q: %v", cloud.Azure.SubnetName, err)
	}

	return nil
}

// createRouteTable will create or update an Azure route table attached to the specified subnet. The call is idempotent.
func createRouteTable(cloud *kubermaticv1.CloudSpec) error {
	routeTablesClient, err := getRouteTablesClient(cloud)
	if err != nil {
		return err
	}

	parameters := network.RouteTable{
		Name:     to.StringPtr(cloud.Azure.RouteTableName),
		Location: to.StringPtr(cloud.Azure.ResourceGroup),
		RouteTablePropertiesFormat: &network.RouteTablePropertiesFormat{
			Subnets: &[]network.Subnet{
				network.Subnet{
					Name: to.StringPtr(cloud.Azure.SubnetName),
				},
			},
		},
	}

	future, err := routeTablesClient.CreateOrUpdate(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName, parameters)
	if err != nil {
		return fmt.Errorf("failed to create or update route table %q: %v", cloud.Azure.RouteTableName, err)
	}

	if err = future.WaitForCompletion(context.TODO(), routeTablesClient.Client); err != nil {
		return fmt.Errorf("failed to create or update route table %q: %v", cloud.Azure.RouteTableName, err)
	}

	return nil
}

func (a *azure) InitializeCloudProvider(cloud *kubermaticv1.CloudSpec, clusterName string) (*kubermaticv1.CloudSpec, error) {
	dc, ok := a.dcs[cloud.DatacenterName]
	if !ok {
		return nil, fmt.Errorf("could not find datacenter %s", cloud.DatacenterName)
	}

	if dc.Spec.Azure == nil {
		return nil, fmt.Errorf("datacenter %q is not a valid Azure datacenter", cloud.DatacenterName)
	}

	if cloud.Azure.ResourceGroup == "" {
		cloud.Azure.ResourceGroup = "cluster-" + clusterName
	}

	if cloud.Azure.VNetName == "" {
		cloud.Azure.VNetName = "cluster-" + clusterName
	}

	if cloud.Azure.SubnetName == "" {
		cloud.Azure.SubnetName = "cluster-" + clusterName
	}

	if cloud.Azure.RouteTableName == "" {
		cloud.Azure.RouteTableName = "cluster-" + clusterName
	}

	if err := createResourceGroup(cloud, dc.Spec.Azure.Location, clusterName); err != nil {
		return nil, err
	}

	if err := createVNet(cloud, clusterName); err != nil {
		return nil, err
	}

	if err := createSubnet(cloud); err != nil {
		return nil, err
	}

	if err := createRouteTable(cloud); err != nil {
		return nil, err
	}

	// TODO create security group

	return cloud, nil
}

func (a *azure) ValidateCloudSpec(cloud *kubermaticv1.CloudSpec) error {
	if cloud.Azure.ResourceGroup != "" {
		rgClient, err := getGroupsClient(cloud)
		if err != nil {
			return err
		}

		if _, err = rgClient.Get(context.TODO(), cloud.Azure.ResourceGroup); err != nil {
			return err
		}
	}

	if cloud.Azure.VNetName != "" {
		vnetClient, err := getNetworksClient(cloud)
		if err != nil {
			return err
		}

		if _, err = vnetClient.Get(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.VNetName, ""); err != nil {
			return err
		}
	}

	if cloud.Azure.SubnetName != "" {
		subnetClient, err := getSubnetsClient(cloud)
		if err != nil {
			return err
		}

		if _, err = subnetClient.Get(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName, ""); err != nil {
			return err
		}
	}

	if cloud.Azure.RouteTableName != "" {
		routeTablesClient, err := getRouteTablesClient(cloud)
		if err != nil {
			return err
		}

		if _, err = routeTablesClient.Get(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName, ""); err != nil {
			return err
		}
	}

	// TODO verify security group

	return nil
}

func getGroupsClient(cloud *kubermaticv1.CloudSpec) (*resources.GroupsClient, error) {
	var err error
	groupsClient := resources.NewGroupsClient(cloud.Azure.SubscriptionID)
	groupsClient.Authorizer, err = auth.NewClientCredentialsConfig(cloud.Azure.ClientID, cloud.Azure.ClientSecret, cloud.Azure.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &groupsClient, nil
}

func getNetworksClient(cloud *kubermaticv1.CloudSpec) (*network.VirtualNetworksClient, error) {
	var err error
	networksClient := network.NewVirtualNetworksClient(cloud.Azure.SubscriptionID)
	networksClient.Authorizer, err = auth.NewClientCredentialsConfig(cloud.Azure.ClientID, cloud.Azure.ClientSecret, cloud.Azure.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &networksClient, nil
}

func getSubnetsClient(cloud *kubermaticv1.CloudSpec) (*network.SubnetsClient, error) {
	var err error
	subnetsClient := network.NewSubnetsClient(cloud.Azure.SubscriptionID)
	subnetsClient.Authorizer, err = auth.NewClientCredentialsConfig(cloud.Azure.ClientID, cloud.Azure.ClientSecret, cloud.Azure.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &subnetsClient, nil
}

func getRouteTablesClient(cloud *kubermaticv1.CloudSpec) (*network.RouteTablesClient, error) {
	var err error
	routeTablesClient := network.NewRouteTablesClient(cloud.Azure.SubscriptionID)
	routeTablesClient.Authorizer, err = auth.NewClientCredentialsConfig(cloud.Azure.ClientID, cloud.Azure.ClientSecret, cloud.Azure.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &routeTablesClient, nil
}
