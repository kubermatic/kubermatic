package azure

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	clusterTagKey = "cluster"

	finalizerSecurityGroup = "kubermatic.io/cleanup-azure-security-group"
	finalizerRouteTable    = "kubermatic.io/cleanup-azure-route-table"
	finalizerSubnet        = "kubermatic.io/cleanup-azure-subnet"
	finalizerVNet          = "kubermatic.io/cleanup-azure-vnet"
	finalizerResourceGroup = "kubermatic.io/cleanup-azure-resource-group"
)

type azure struct{}

// New returns a new Azure provider.
func New() provider.CloudProvider {
	return &azure{}
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
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = kuberneteshelper.RemoveFinalizer(cluster.Finalizers, finalizerSecurityGroup)
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
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = kuberneteshelper.RemoveFinalizer(cluster.Finalizers, finalizerRouteTable)
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
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = kuberneteshelper.RemoveFinalizer(cluster.Finalizers, finalizerSubnet)
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

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = kuberneteshelper.RemoveFinalizer(cluster.Finalizers, finalizerVNet)
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

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = kuberneteshelper.RemoveFinalizer(cluster.Finalizers, finalizerResourceGroup)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// ensureResourceGroup will create or update an Azure resource group. The call is idempotent.
func ensureResourceGroup(cloud *kubermaticv1.CloudSpec, clusterName string) error {
	groupsClient, err := getGroupsClient(cloud)
	if err != nil {
		return err
	}

	parameters := resources.Group{
		Name:     to.StringPtr(cloud.Azure.ResourceGroup),
		Location: to.StringPtr(cloud.Azure.Location),
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
func ensureSecurityGroup(cloud *kubermaticv1.CloudSpec, clusterName string) error {
	sgClient, err := getSecurityGroupsClient(cloud)
	if err != nil {
		return err
	}

	parameters := network.SecurityGroup{
		Name:     to.StringPtr(cloud.Azure.SecurityGroup),
		Location: to.StringPtr(cloud.Azure.Location),
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
	if _, err = sgClient.CreateOrUpdate(context.TODO(), cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup, parameters); err != nil {
		return fmt.Errorf("failed to create or update resource group %q: %v", cloud.Azure.ResourceGroup, err)
	}

	return nil
}

// ensureVNet will create or update an Azure virtual network in the specified resource group. The call is idempotent.
func ensureVNet(cloud *kubermaticv1.CloudSpec, clusterName string) error {
	networksClient, err := getNetworksClient(cloud)
	if err != nil {
		return err
	}

	parameters := network.VirtualNetwork{
		Name:     to.StringPtr(cloud.Azure.VNetName),
		Location: to.StringPtr(cloud.Azure.Location),
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
func ensureRouteTable(cloud *kubermaticv1.CloudSpec) error {
	routeTablesClient, err := getRouteTablesClient(cloud)
	if err != nil {
		return err
	}

	parameters := network.RouteTable{
		Name:     to.StringPtr(cloud.Azure.RouteTableName),
		Location: to.StringPtr(cloud.Azure.Location),
		RouteTablePropertiesFormat: &network.RouteTablePropertiesFormat{
			Subnets: &[]network.Subnet{
				{
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
	if cluster.Spec.Cloud.Azure.ResourceGroup == "" {
		cluster.Spec.Cloud.Azure.ResourceGroup = "cluster-" + cluster.Name

		glog.Infof("cluster %q: ensuring resource group %q", cluster.Name, cluster.Spec.Cloud.Azure.ResourceGroup)
		if err := ensureResourceGroup(cluster.Spec.Cloud, cluster.Name); err != nil {
			return cluster, err
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = append(cluster.Finalizers, finalizerResourceGroup)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Azure.VNetName == "" {
		cluster.Spec.Cloud.Azure.VNetName = "cluster-" + cluster.Name

		glog.Infof("cluster %q: ensuring vnet %q", cluster.Name, cluster.Spec.Cloud.Azure.VNetName)
		if err := ensureVNet(cluster.Spec.Cloud, cluster.Name); err != nil {
			return cluster, err
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = append(cluster.Finalizers, finalizerVNet)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Azure.SubnetName == "" {
		cluster.Spec.Cloud.Azure.SubnetName = "cluster-" + cluster.Name

		glog.Infof("cluster %q: ensuring subnet %q", cluster.Name, cluster.Spec.Cloud.Azure.SubnetName)
		if err := ensureSubnet(cluster.Spec.Cloud); err != nil {
			return cluster, err
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = append(cluster.Finalizers, finalizerSubnet)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Azure.RouteTableName == "" {
		cluster.Spec.Cloud.Azure.RouteTableName = "cluster-" + cluster.Name

		glog.Infof("cluster %q: ensuring route table %q", cluster.Name, cluster.Spec.Cloud.Azure.RouteTableName)
		if err := ensureRouteTable(cluster.Spec.Cloud); err != nil {
			return cluster, err
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = append(cluster.Finalizers, finalizerRouteTable)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Azure.SecurityGroup == "" {
		cluster.Spec.Cloud.Azure.SecurityGroup = "cluster-" + cluster.Name

		glog.Infof("cluster %q: ensuring security group %q", cluster.Name, cluster.Spec.Cloud.Azure.SecurityGroup)
		if err := ensureSecurityGroup(cluster.Spec.Cloud, cluster.Name); err != nil {
			return cluster, err
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = append(cluster.Finalizers, finalizerSecurityGroup)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
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
