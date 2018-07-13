package openstack

import (
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"

	"github.com/gophercloud/gophercloud"
	osflavors "github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	osprojects "github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

const (
	securityGroupCleanupFinalizer = "kubermatic.io/cleanup-openstack-security-group"
	networkCleanupFinalizer       = "kubermatic.io/cleanup-openstack-network"
)

// Provider is a struct that implements CloudProvider interface
type Provider struct {
	dcs map[string]provider.DatacenterMeta
}

// NewCloudProvider creates a new openstack provider.
func NewCloudProvider(dcs map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &Provider{
		dcs: dcs,
	}
}

// ValidateCloudSpec validates the given CloudSpec
func (os *Provider) ValidateCloudSpec(spec *kubermaticv1.CloudSpec) error {
	netClient, err := os.OpenstackProvider(spec.Openstack.Username, spec.Openstack.Password, spec.Openstack.Domain, spec.DatacenterName)
	if err != nil {
		return fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if spec.Openstack.SecurityGroups != "" {
		if err := validateSecurityGroupsExist(netClient, strings.Split(spec.Openstack.SecurityGroups, ",")); err != nil {
			return err
		}
	}

	if spec.Openstack.Network != "" {
		_, err := getNetworkByName(netClient, spec.Openstack.Network, false)
		if err != nil {
			return fmt.Errorf("failed to get network %q: %v", spec.Openstack.Network, err)
		}
	}

	if spec.Openstack.FloatingIPPool != "" {
		_, err := getNetworkByName(netClient, spec.Openstack.FloatingIPPool, true)
		if err != nil {
			return fmt.Errorf("failed to get floating ip pool %q: %v", spec.Openstack.FloatingIPPool, err)
		}
	}

	return nil
}

// InitializeCloudProvider initializes a cluster, in particular
// creates security group and network configuration
func (os *Provider) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	dc, found := os.dcs[cluster.Spec.Cloud.DatacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, fmt.Errorf("invalid datacenter %q", cluster.Spec.Cloud.DatacenterName)
	}

	spec := cluster.Spec.Cloud
	netClient, err := os.OpenstackProvider(spec.Openstack.Username, spec.Openstack.Password, spec.Openstack.Domain, spec.DatacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if cluster.Spec.Cloud.Openstack.FloatingIPPool == "" {
		extNetwork, err := getExternalNetwork(netClient)
		if err != nil {
			return nil, err
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.FloatingIPPool = extNetwork.Name
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Openstack.SecurityGroups == "" {
		g, err := createKubermaticSecurityGroup(netClient, cluster.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic security group: %v", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.SecurityGroups = g.Name
			cluster.Finalizers = append(cluster.Finalizers, securityGroupCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Openstack.Network == "" {
		network, err := createKubermaticNetwork(netClient, cluster.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic network: %v", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.Network = network.Name
		})
		if err != nil {
			return nil, err
		}

		subnet, err := createKubermaticSubnet(netClient, cluster.Name, network.ID, dc.Spec.Openstack.DNSServers)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic subnet: %v", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.SubnetID = subnet.ID
		})
		if err != nil {
			return nil, err
		}

		router, err := createKubermaticRouter(netClient, cluster.Name, cluster.Spec.Cloud.Openstack.FloatingIPPool)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic router: %v", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.RouterID = router.ID
		})
		if err != nil {
			return nil, err
		}

		if _, err = attachSubnetToRouter(netClient, subnet.ID, router.ID); err != nil {
			return nil, fmt.Errorf("failed to attach subnet to router: %v", err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = append(cluster.Finalizers, networkCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// CleanUpCloudProvider does the clean-up in particular:
// removes security group and network configuration
func (os *Provider) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	spec := cluster.Spec.Cloud
	netClient, err := os.OpenstackProvider(spec.Openstack.Username, spec.Openstack.Password, spec.Openstack.Domain, spec.DatacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if kubernetes.HasFinalizer(cluster, securityGroupCleanupFinalizer) {
		for _, g := range strings.Split(cluster.Spec.Cloud.Openstack.SecurityGroups, ",") {
			if err := deleteSecurityGroup(netClient, strings.TrimSpace(g)); err != nil {
				return nil, fmt.Errorf("failed to delete security group %q: %v", g, err)
			}
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = kubernetes.RemoveFinalizer(cluster.Finalizers, securityGroupCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if kubernetes.HasFinalizer(cluster, networkCleanupFinalizer) {
		if _, err = detachSubnetFromRouter(netClient, cluster.Spec.Cloud.Openstack.SubnetID, cluster.Spec.Cloud.Openstack.RouterID); err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); !ok {
				return nil, fmt.Errorf("failed to detach subnet from router: %v", err)
			}
		}

		if err = deleteNetworkByName(netClient, cluster.Spec.Cloud.Openstack.Network); err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); !ok {
				return nil, fmt.Errorf("failed delete network %q: %v", cluster.Spec.Cloud.Openstack.Network, err)
			}
		}

		if err = deleteRouter(netClient, cluster.Spec.Cloud.Openstack.RouterID); err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); !ok {
				return nil, fmt.Errorf("failed delete router %q: %v", cluster.Spec.Cloud.Openstack.RouterID, err)
			}
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = kubernetes.RemoveFinalizer(cluster.Finalizers, networkCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// GetFlavors lists available flavors for the given CloudSpec.DatacenterName and OpenstackSpec.Region
func (os *Provider) GetFlavors(serviceClient *gophercloud.ServiceClient, datacenterName string) ([]osflavors.Flavor, provider.DatacenterMeta, error) {

	dc, found := os.dcs[datacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, provider.DatacenterMeta{}, fmt.Errorf("invalid datacenter %q", datacenterName)
	}

	flavors, err := getFlavors(serviceClient.ProviderClient, dc.Spec.Openstack.Region)
	if err != nil {
		return nil, provider.DatacenterMeta{}, err
	}

	return flavors, dc, nil
}

// GetTenants lists all available tenents for the given CloudSpec.DatacenterName
func (os *Provider) GetTenants(serviceClient *gophercloud.ServiceClient, datacenterName string) ([]osprojects.Project, error) {
	dc, found := os.dcs[datacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, fmt.Errorf("invalid datacenter %q", datacenterName)
	}

	tenants, err := getTenants(serviceClient.ProviderClient, dc.Spec.Openstack.Region)
	if err != nil {
		return nil, fmt.Errorf("couldn't get tenants for region %s: %v", dc.Spec.Openstack.Region, err)
	}

	return tenants, nil
}

// GetNetworks lists all available networks for the given CloudSpec.DatacenterName
func (os *Provider) GetNetworks(serviceClient *gophercloud.ServiceClient) ([]NetworkWithExternalExt, error) {
	networks, err := getAllNetworks(serviceClient)
	if err != nil {
		return nil, fmt.Errorf("couldn't get networks: %v", err)
	}

	return networks, nil
}

// GetSubnetIDs list all available subnet ids fot a given CloudSpec
func (os *Provider) GetSubnetIDs(serviceClient *gophercloud.ServiceClient) ([]Subnet, error) {
	allSubnets, err := getAllSubnets(serviceClient)
	if err != nil {
		return nil, err
	}
	subnets := []Subnet{}
	for _, subnet := range allSubnets {
		subnets = append(subnets, Subnet{
			Name: subnet.Name,
			ID:   subnet.ID,
		})
	}
	return subnets, nil
}

// OpenstackProvider returns gophercloud.ServiceClient
func (os *Provider) OpenstackProvider(user, pass, domain, datacenterName string) (*gophercloud.ServiceClient, error) {
	dc, found := os.dcs[datacenterName]
	if !found || dc.Spec.Openstack == nil {
		return nil, fmt.Errorf("invalid datacenter %q", datacenterName)
	}
	client, err := getServiceClient(user, pass, domain, "", dc.Spec.Openstack.Region, dc.Spec.Openstack.AuthURL)
	if err != nil {
		return nil, err
	}
	return client, nil
}
