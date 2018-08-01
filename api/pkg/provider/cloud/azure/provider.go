package azure

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	clusterTagKey = "cluster"

	finalizerSecurityGroup   = "kubermatic.io/cleanup-azure-security-group"
	finalizerRouteTable      = "kubermatic.io/cleanup-azure-route-table"
	finalizerSubnet          = "kubermatic.io/cleanup-azure-subnet"
	finalizerVNet            = "kubermatic.io/cleanup-azure-vnet"
	finalizerResourceGroup   = "kubermatic.io/cleanup-azure-resource-group"
	finalizerAvailabilitySet = "kubermatic.io/cleanup-azure-availability-set"
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
		return err
	}

	if err = deleteSubnetFuture.WaitForCompletion(context.TODO(), subnetsClient.Client); err != nil {
		return err
	}

	return nil
}

func deleteAvailabilitySet(cloud *kubermaticv1.CloudSpec) error {
	asClient, err := getAvailabilitySetClient(cloud)
	if err != nil {
		return err
	}

	_, err = asClient.Delete(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.AvailabilitySet)
	return err
}

func deleteVNet(cloud *kubermaticv1.CloudSpec) error {
	networksClient, err := getNetworksClient(cloud)
	if err != nil {
		return err
	}

	deleteVNetFuture, err := networksClient.Delete(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.VNetName)
	if err != nil {
		return err
	}

	if err = deleteVNetFuture.WaitForCompletion(context.TODO(), networksClient.Client); err != nil {
		return err
	}

	return nil
}

func deleteResourceGroup(cloud *kubermaticv1.CloudSpec) error {
	groupsClient, err := getGroupsClient(cloud)
	if err != nil {
		return err
	}

	// We're doing a Get to see if its already gone or not.
	// We could also directly call delete but the error response would need to be unpacked twice to get the correct error message.
	// Doing a get is simpler.
	if _, err := groupsClient.Get(context.TODO(), cloud.Azure.ResourceGroup); err != nil {
		return err
	}

	future, err := groupsClient.Delete(context.TODO(), cloud.Azure.ResourceGroup)
	if err != nil {
		return err
	}

	if err = future.WaitForCompletion(context.TODO(), groupsClient.Client); err != nil {
		return err
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
		return err
	}

	if err = future.WaitForCompletion(context.TODO(), routeTablesClient.Client); err != nil {
		return err
	}

	return nil
}

func deleteSecurityGroup(cloud *kubermaticv1.CloudSpec) error {
	securityGroupsClient, err := getSecurityGroupsClient(cloud)
	if err != nil {
		return err
	}

	future, err := securityGroupsClient.Delete(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup)
	if err != nil {
		return err
	}

	if err = future.WaitForCompletion(context.TODO(), securityGroupsClient.Client); err != nil {
		return err
	}

	return nil
}

func (a *azure) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var err error
	if kuberneteshelper.HasFinalizer(cluster, finalizerSecurityGroup) {
		glog.Infof("cluster %q: deleting security group %q", cluster.Name, cluster.Spec.Cloud.Azure.SecurityGroup)
		if err := deleteSecurityGroup(cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete security group %q: %v", cluster.Spec.Cloud.Azure.SecurityGroup, err)
			}
		}
		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Finalizers = kuberneteshelper.RemoveFinalizer(updatedCluster.Finalizers, finalizerSecurityGroup)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, finalizerRouteTable) {
		glog.Infof("cluster %q: deleting route table %q", cluster.Name, cluster.Spec.Cloud.Azure.RouteTableName)
		if err := deleteRouteTable(cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete route table %q: %v", cluster.Spec.Cloud.Azure.RouteTableName, err)
			}
		}
		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Finalizers = kuberneteshelper.RemoveFinalizer(updatedCluster.Finalizers, finalizerRouteTable)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, finalizerSubnet) {
		glog.Infof("cluster %q: deleting subnet %q", cluster.Name, cluster.Spec.Cloud.Azure.SubnetName)
		if err := deleteSubnet(cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete sub-network %q: %v", cluster.Spec.Cloud.Azure.SubnetName, err)
			}
		}
		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Finalizers = kuberneteshelper.RemoveFinalizer(updatedCluster.Finalizers, finalizerSubnet)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, finalizerVNet) {
		glog.Infof("cluster %q: deleting vnet %q", cluster.Name, cluster.Spec.Cloud.Azure.VNetName)
		if err := deleteVNet(cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete virtual network %q: %v", cluster.Spec.Cloud.Azure.VNetName, err)
			}
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Finalizers = kuberneteshelper.RemoveFinalizer(updatedCluster.Finalizers, finalizerVNet)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, finalizerResourceGroup) {
		glog.Infof("cluster %q: deleting resource group %q", cluster.Name, cluster.Spec.Cloud.Azure.ResourceGroup)
		if err := deleteResourceGroup(cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete resource group %q: %v", cluster.Spec.Cloud.Azure.ResourceGroup, err)
			}
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Finalizers = kuberneteshelper.RemoveFinalizer(updatedCluster.Finalizers, finalizerResourceGroup)
		})
		if err != nil {
			return nil, err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, finalizerAvailabilitySet) {
		glog.Infof("cluster %q: deleting availability set %q", cluster.Name, cluster.Spec.Cloud.Azure.AvailabilitySet)
		if err := deleteAvailabilitySet(cluster.Spec.Cloud); err != nil {
			if detErr, ok := err.(autorest.DetailedError); !ok || detErr.StatusCode != http.StatusNotFound {
				return cluster, fmt.Errorf("failed to delete availability set %q: %v", cluster.Spec.Cloud.Azure.AvailabilitySet, err)
			}
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Finalizers = kuberneteshelper.RemoveFinalizer(updatedCluster.Finalizers, finalizerAvailabilitySet)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// ensureResourceGroup will create or update an Azure resource group. The call is idempotent.
func ensureResourceGroup(cloud *kubermaticv1.CloudSpec, location string, clusterName string) error {
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

// ensureSecurityGroup will create or update an Azure security group. The call is idempotent.
func ensureSecurityGroup(cloud *kubermaticv1.CloudSpec, location string, clusterName string) error {
	sgClient, err := getSecurityGroupsClient(cloud)
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
				network.Subnet{
					Name: to.StringPtr(cloud.Azure.SubnetName),
					ID:   to.StringPtr(assembleSubnetID(cloud)),
				},
			},
			// inbound
			SecurityRules: &[]network.SecurityRule{
				network.SecurityRule{
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
				network.SecurityRule{
					Name: to.StringPtr("kubelet"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionInbound,
						Protocol:                 network.SecurityRuleProtocolTCP,
						SourceAddressPrefix:      to.StringPtr("*"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("10250"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(101),
					},
				},
				network.SecurityRule{
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
				network.SecurityRule{
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
				network.SecurityRule{
					Name: to.StringPtr("deny_all"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionInbound,
						Protocol:                 "*",
						SourceAddressPrefix:      to.StringPtr("*"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						Access:   network.SecurityRuleAccessDeny,
						Priority: to.Int32Ptr(800),
					},
				},
				// outbound
				network.SecurityRule{
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
	if _, err = sgClient.CreateOrUpdate(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup, parameters); err != nil {
		return fmt.Errorf("failed to create or update resource group %q: %v", cloud.Azure.ResourceGroup, err)
	}

	return nil
}

// ensureVNet will create or update an Azure virtual network in the specified resource group. The call is idempotent.
func ensureVNet(cloud *kubermaticv1.CloudSpec, location string, clusterName string) error {
	networksClient, err := getNetworksClient(cloud)
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

	future, err := networksClient.CreateOrUpdate(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.VNetName, parameters)
	if err != nil {
		return fmt.Errorf("failed to create or update virtual network %q: %v", cloud.Azure.VNetName, err)
	}

	if err = future.WaitForCompletion(context.TODO(), networksClient.Client); err != nil {
		return fmt.Errorf("failed to create or update virtual network %q: %v", cloud.Azure.VNetName, err)
	}

	return nil
}

// ensureSubnet will create or update an Azure subnetwork in the specified vnet. The call is idempotent.
func ensureSubnet(cloud *kubermaticv1.CloudSpec) error {
	subnetsClient, err := getSubnetsClient(cloud)
	if err != nil {
		return err
	}

	parameters := network.Subnet{
		Name: to.StringPtr(cloud.Azure.SubnetName),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr("10.0.0.0/16"),
		},
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

// ensureRouteTable will create or update an Azure route table attached to the specified subnet. The call is idempotent.
func ensureRouteTable(cloud *kubermaticv1.CloudSpec, location string) error {
	routeTablesClient, err := getRouteTablesClient(cloud)
	if err != nil {
		return err
	}

	parameters := network.RouteTable{
		Name:     to.StringPtr(cloud.Azure.RouteTableName),
		Location: to.StringPtr(location),
		RouteTablePropertiesFormat: &network.RouteTablePropertiesFormat{
			Subnets: &[]network.Subnet{
				network.Subnet{
					Name: to.StringPtr(cloud.Azure.SubnetName),
					ID:   to.StringPtr(assembleSubnetID(cloud)),
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

func (a *azure) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var err error
	dc, ok := a.dcs[cluster.Spec.Cloud.DatacenterName]
	if !ok {
		return nil, fmt.Errorf("could not find datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	if dc.Spec.Azure == nil {
		return nil, fmt.Errorf("datacenter %q is not a valid Azure datacenter", cluster.Spec.Cloud.DatacenterName)
	}

	location := dc.Spec.Azure.Location

	if cluster.Spec.Cloud.Azure.ResourceGroup == "" {
		cluster.Spec.Cloud.Azure.ResourceGroup = "cluster-" + cluster.Name

		glog.Infof("cluster %q: ensuring resource group %q", cluster.Name, cluster.Spec.Cloud.Azure.ResourceGroup)
		if err = ensureResourceGroup(cluster.Spec.Cloud, location, cluster.Name); err != nil {
			return cluster, err
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.ResourceGroup = cluster.Spec.Cloud.Azure.ResourceGroup
			updatedCluster.Finalizers = append(updatedCluster.Finalizers, finalizerResourceGroup)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Azure.VNetName == "" {
		cluster.Spec.Cloud.Azure.VNetName = "cluster-" + cluster.Name

		glog.Infof("cluster %q: ensuring vnet %q", cluster.Name, cluster.Spec.Cloud.Azure.VNetName)
		if err = ensureVNet(cluster.Spec.Cloud, location, cluster.Name); err != nil {
			return cluster, err
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.VNetName = cluster.Spec.Cloud.Azure.VNetName
			updatedCluster.Finalizers = append(updatedCluster.Finalizers, finalizerVNet)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Azure.SubnetName == "" {
		cluster.Spec.Cloud.Azure.SubnetName = "cluster-" + cluster.Name

		glog.Infof("cluster %q: ensuring subnet %q", cluster.Name, cluster.Spec.Cloud.Azure.SubnetName)
		if err = ensureSubnet(cluster.Spec.Cloud); err != nil {
			return cluster, err
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.SubnetName = cluster.Spec.Cloud.Azure.SubnetName
			updatedCluster.Finalizers = append(updatedCluster.Finalizers, finalizerSubnet)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Azure.RouteTableName == "" {
		cluster.Spec.Cloud.Azure.RouteTableName = "cluster-" + cluster.Name

		glog.Infof("cluster %q: ensuring route table %q", cluster.Name, cluster.Spec.Cloud.Azure.RouteTableName)
		if err = ensureRouteTable(cluster.Spec.Cloud, location); err != nil {
			return cluster, err
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.RouteTableName = cluster.Spec.Cloud.Azure.RouteTableName
			updatedCluster.Finalizers = append(updatedCluster.Finalizers, finalizerRouteTable)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Azure.SecurityGroup == "" {
		cluster.Spec.Cloud.Azure.SecurityGroup = "cluster-" + cluster.Name

		glog.Infof("cluster %q: ensuring security group %q", cluster.Name, cluster.Spec.Cloud.Azure.SecurityGroup)
		if err = ensureSecurityGroup(cluster.Spec.Cloud, location, cluster.Name); err != nil {
			return cluster, err
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.SecurityGroup = cluster.Spec.Cloud.Azure.SecurityGroup
			updatedCluster.Finalizers = append(updatedCluster.Finalizers, finalizerSecurityGroup)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Azure.AvailabilitySet == "" {
		asName := "cluster-" + cluster.Name
		glog.Infof("cluster %q: ensuring AvailabilitySet %q", cluster.Name, asName)

		if err := ensureAvailabilitySet(asName, location, cluster.Spec.Cloud); err != nil {
			return nil, fmt.Errorf("failed to ensure AvailabilitySet exists: %v", err)
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.AvailabilitySet = asName
			updatedCluster.Finalizers = append(updatedCluster.Finalizers, finalizerAvailabilitySet)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Azure.AvailabilitySet == "" {
		asName := "cluster-" + cluster.Name
		glog.Infof("cluster %q: ensuring AvailabilitySet %q", cluster.Name, asName)

		if err := ensureAvailabilitySet(asName, location, cluster.Spec.Cloud); err != nil {
			return nil, fmt.Errorf("failed to ensure AvailabilitySet exists: %v", err)
		}

		cluster, err = update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.AvailabilitySet = asName
			updatedCluster.Finalizers = append(updatedCluster.Finalizers, finalizerAvailabilitySet)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

func ensureAvailabilitySet(name, location string, cloud *kubermaticv1.CloudSpec) error {
	client, err := getAvailabilitySetClient(cloud)
	if err != nil {
		return err
	}

	as := compute.AvailabilitySet{
		Name:     to.StringPtr(name),
		Location: to.StringPtr(location),
		Sku: &compute.Sku{
			Name: to.StringPtr("Aligned"),
		},
		AvailabilitySetProperties: &compute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  to.Int32Ptr(3),
			PlatformUpdateDomainCount: to.Int32Ptr(20),
		},
	}

	_, err = client.CreateOrUpdate(context.TODO(), cloud.Azure.ResourceGroup, name, as)
	return err
}

func (a *azure) DefaultCloudSpec(cloud *kubermaticv1.CloudSpec) error {
	return nil
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

	if cloud.Azure.SecurityGroup != "" {
		sgClient, err := getSecurityGroupsClient(cloud)
		if err != nil {
			return err
		}

		if _, err = sgClient.Get(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup, ""); err != nil {
			return err
		}
	}

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

func getSecurityGroupsClient(cloud *kubermaticv1.CloudSpec) (*network.SecurityGroupsClient, error) {
	var err error
	securityGroupsClient := network.NewSecurityGroupsClient(cloud.Azure.SubscriptionID)
	securityGroupsClient.Authorizer, err = auth.NewClientCredentialsConfig(cloud.Azure.ClientID, cloud.Azure.ClientSecret, cloud.Azure.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &securityGroupsClient, nil
}

func getAvailabilitySetClient(cloud *kubermaticv1.CloudSpec) (*compute.AvailabilitySetsClient, error) {
	var err error
	asClient := compute.NewAvailabilitySetsClient(cloud.Azure.SubscriptionID)
	asClient.Authorizer, err = auth.NewClientCredentialsConfig(cloud.Azure.ClientID, cloud.Azure.ClientSecret, cloud.Azure.TenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %s", err.Error())
	}

	return &asClient, nil
}
