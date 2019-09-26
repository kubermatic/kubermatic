package openstack

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"
	osflavors "github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	osprojects "github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	ossecuritygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	osecuritygrouprules "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	osnetworks "github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	ossubnets "github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

const (
	// SecurityGroupCleanupFinalizer will instruct the deletion of the security group
	SecurityGroupCleanupFinalizer = "kubermatic.io/cleanup-openstack-security-group"
	// OldNetworkCleanupFinalizer will instruct the deletion of all network components. Router, Network, Subnet
	// Deprecated: Got splitted into dedicated finalizers
	OldNetworkCleanupFinalizer = "kubermatic.io/cleanup-openstack-network"

	// NetworkCleanupFinalizer will instruct the deletion of the network
	NetworkCleanupFinalizer = "kubermatic.io/cleanup-openstack-network-v2"
	// SubnetCleanupFinalizer will instruct the deletion of the subnet
	SubnetCleanupFinalizer = "kubermatic.io/cleanup-openstack-subnet-v2"
	// RouterCleanupFinalizer will instruct the deletion of the router
	RouterCleanupFinalizer = "kubermatic.io/cleanup-openstack-router-v2"
	// RouterSubnetLinkCleanupFinalizer will instruct the deletion of the link between the router and the subnet
	RouterSubnetLinkCleanupFinalizer = "kubermatic.io/cleanup-openstack-router-subnet-link-v2"
)

// Provider is a struct that implements CloudProvider interface
type Provider struct {
	dc                *kubermaticv1.DatacenterSpecOpenstack
	secretKeySelector provider.SecretKeySelectorValueFunc
}

// NewCloudProvider creates a new openstack provider.
func NewCloudProvider(dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc) (*Provider, error) {
	if dc.Spec.Openstack == nil {
		return nil, errors.New("datacenter is not an Openstack datacenter")
	}
	return &Provider{
		dc:                dc.Spec.Openstack,
		secretKeySelector: secretKeyGetter,
	}, nil
}

// DefaultCloudSpec adds defaults to the cloud spec
func (os *Provider) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

// ValidateCloudSpec validates the given CloudSpec
func (os *Provider) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	netClient, err := os.getNetClient(spec)
	if err != nil {
		return fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if spec.Openstack.SecurityGroups != "" {
		if err := validateSecurityGroupsExist(netClient, strings.Split(spec.Openstack.SecurityGroups, ",")); err != nil {
			return err
		}
	}

	if spec.Openstack.Network != "" {
		network, err := getNetworkByName(netClient, spec.Openstack.Network, false)
		if err != nil {
			return fmt.Errorf("failed to get network %q: %v", spec.Openstack.Network, err)
		}

		// If we're going to create a subnet in an existing network,
		// let's check whether any existing subnets collide with our range.
		if spec.Openstack.SubnetID == "" {
			if err = validateExistingSubnetOverlap(network.ID, netClient); err != nil {
				return err
			}
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

// validateExistingSubnetOverlap checks whether any subnets in the given network overlap with the default subnet CIDR
func validateExistingSubnetOverlap(networkID string, netClient *gophercloud.ServiceClient) error {
	_, defaultCIDR, err := net.ParseCIDR(subnetCIDR)
	if err != nil {
		return err
	}

	pager := ossubnets.List(netClient, ossubnets.ListOpts{NetworkID: networkID})
	return pager.EachPage(func(page pagination.Page) (bool, error) {
		subnets, extractErr := ossubnets.ExtractSubnets(page)
		if extractErr != nil {
			return false, extractErr
		}

		for _, sn := range subnets {
			_, currentCIDR, parseErr := net.ParseCIDR(sn.CIDR)
			if parseErr != nil {
				return false, parseErr
			}

			// do the CIDRs overlap?
			if currentCIDR.Contains(defaultCIDR.IP) || defaultCIDR.Contains(currentCIDR.IP) {
				return false, fmt.Errorf("existing subnetwork %q holds a CIDR %q which overlaps with default CIDR %q", sn.Name, sn.CIDR, subnetCIDR)
			}
		}

		return true, nil
	})
}

// InitializeCloudProvider initializes a cluster, in particular
// creates security group and network configuration
func (os *Provider) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	netClient, err := os.getNetClient(cluster.Spec.Cloud)
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
			// We're just searching for the floating ip pool here & don't create anything. Thus no need to create a finalizer
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Openstack.SecurityGroups == "" {
		secGroupName, err := createKubermaticSecurityGroup(netClient, cluster.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic security group: %v", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.SecurityGroups = secGroupName
			kubernetes.AddFinalizer(cluster, SecurityGroupCleanupFinalizer)
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
			kubernetes.AddFinalizer(cluster, NetworkCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	network, err := getNetworkByName(netClient, cluster.Spec.Cloud.Openstack.Network, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get network '%s': %v", cluster.Spec.Cloud.Openstack.Network, err)
	}

	if cluster.Spec.Cloud.Openstack.SubnetID == "" {
		subnet, err := createKubermaticSubnet(netClient, cluster.Name, network.ID, os.dc.DNSServers)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic subnet: %v", err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.SubnetID = subnet.ID
			kubernetes.AddFinalizer(cluster, SubnetCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if cluster.Spec.Cloud.Openstack.RouterID == "" {
		// Check if the subnet has already a router
		routerID, err := getRouterIDForSubnet(netClient, cluster.Spec.Cloud.Openstack.SubnetID, network.ID)
		if err != nil {
			if err == errNotFound {
				// No Router exists -> Create a router
				router, err := createKubermaticRouter(netClient, cluster.Name, cluster.Spec.Cloud.Openstack.FloatingIPPool)
				if err != nil {
					return nil, fmt.Errorf("failed to create the kubermatic router: %v", err)
				}
				cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
					cluster.Spec.Cloud.Openstack.RouterID = router.ID
					kubernetes.AddFinalizer(cluster, RouterCleanupFinalizer)
				})
				if err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("failed to verify that the subnet '%s' has a router attached: %v", cluster.Spec.Cloud.Openstack.SubnetID, err)
			}
		} else {
			// A router already exists -> Reuse it but don't clean it up
			cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
				cluster.Spec.Cloud.Openstack.RouterID = routerID
			})
			if err != nil {
				return nil, err
			}
		}
	}

	// If we created the subnet, but have not created the router-subnet-link finalizer, we need to attach the subnet to the router
	// Otherwise the vm's won't have connectivity
	if kubernetes.HasFinalizer(cluster, SubnetCleanupFinalizer) && !kubernetes.HasFinalizer(cluster, RouterSubnetLinkCleanupFinalizer) {
		if _, err = attachSubnetToRouter(netClient, cluster.Spec.Cloud.Openstack.SubnetID, cluster.Spec.Cloud.Openstack.RouterID); err != nil {
			return nil, fmt.Errorf("failed to attach subnet to router: %v", err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.AddFinalizer(cluster, RouterSubnetLinkCleanupFinalizer)
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
	netClient, err := os.getNetClient(cluster.Spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if kubernetes.HasFinalizer(cluster, SecurityGroupCleanupFinalizer) {
		for _, g := range strings.Split(cluster.Spec.Cloud.Openstack.SecurityGroups, ",") {
			if err := deleteSecurityGroup(netClient, strings.TrimSpace(g)); err != nil {
				return nil, fmt.Errorf("failed to delete security group %q: %v", g, err)
			}
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.RemoveFinalizer(cluster, SecurityGroupCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if kubernetes.HasFinalizer(cluster, RouterSubnetLinkCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if _, err = detachSubnetFromRouter(netClient, cluster.Spec.Cloud.Openstack.SubnetID, cluster.Spec.Cloud.Openstack.RouterID); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to detach subnet from router: %v", err)
			}
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.RemoveFinalizer(cluster, RouterSubnetLinkCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if kubernetes.HasFinalizer(cluster, SubnetCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if err := deleteSubnet(netClient, cluster.Spec.Cloud.Openstack.SubnetID); err != nil {
			return nil, fmt.Errorf("failed to delete subnet '%s': %v", cluster.Spec.Cloud.Openstack.SubnetID, err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.RemoveFinalizer(cluster, SubnetCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if kubernetes.HasFinalizer(cluster, NetworkCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if err = deleteNetworkByName(netClient, cluster.Spec.Cloud.Openstack.Network); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete network '%s': %v", cluster.Spec.Cloud.Openstack.Network, err)
			}
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.RemoveFinalizer(cluster, NetworkCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if kubernetes.HasFinalizer(cluster, RouterCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if err = deleteRouter(netClient, cluster.Spec.Cloud.Openstack.RouterID); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete router '%s': %v", cluster.Spec.Cloud.Openstack.RouterID, err)
			}
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.RemoveFinalizer(cluster, RouterCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	if kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.RemoveFinalizer(cluster, OldNetworkCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// GetFlavors lists available flavors for the given CloudSpec.DatacenterName and OpenstackSpec.Region
func (os *Provider) GetFlavors(cloud kubermaticv1.CloudSpec) ([]osflavors.Flavor, error) {
	authClient, err := os.getAuthClient(cloud)
	if err != nil {
		return nil, err
	}
	flavors, err := getFlavors(authClient, os.dc.Region)
	if err != nil {
		return nil, err
	}

	return flavors, nil
}

// GetTenants lists all available tenents for the given CloudSpec.DatacenterName
func (os *Provider) GetTenants(cloud kubermaticv1.CloudSpec) ([]osprojects.Project, error) {
	authClient, err := os.getAuthClient(cloud)
	if err != nil {
		return nil, fmt.Errorf("couldn't get auth client: %v", err)
	}

	region := os.dc.Region
	tenants, err := getTenants(authClient, region)
	if err != nil {
		return nil, fmt.Errorf("couldn't get tenants for region %s: %v", region, err)
	}

	return tenants, nil
}

// GetNetworks lists all available networks for the given CloudSpec.DatacenterName
func (os *Provider) GetNetworks(cloud kubermaticv1.CloudSpec) ([]NetworkWithExternalExt, error) {
	authClient, err := os.getNetClient(cloud)
	if err != nil {
		return nil, fmt.Errorf("couldn't get auth client: %v", err)
	}

	networks, err := getAllNetworks(authClient, osnetworks.ListOpts{})
	if err != nil {
		return nil, fmt.Errorf("couldn't get networks: %v", err)
	}

	return networks, nil
}

// GetSecurityGroups lists all available security groups for the given CloudSpec.DatacenterName
func (os *Provider) GetSecurityGroups(cloud kubermaticv1.CloudSpec) ([]ossecuritygroups.SecGroup, error) {
	netClient, err := os.getNetClient(cloud)
	if err != nil {
		return nil, fmt.Errorf("couldn't get auth client: %v", err)
	}

	page, err := ossecuritygroups.List(netClient, ossecuritygroups.ListOpts{}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list security groups: %v", err)
	}
	secGroups, err := ossecuritygroups.ExtractGroups(page)
	if err != nil {
		return nil, fmt.Errorf("failed to extract security groups: %v", err)
	}
	return secGroups, nil
}

func (os *Provider) getAuthClient(cloud kubermaticv1.CloudSpec) (*gophercloud.ProviderClient, error) {
	creds, err := GetCredentialsForCluster(cloud, os.secretKeySelector)
	if err != nil {
		return nil, err
	}

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: os.dc.AuthURL,
		Username:         creds.Username,
		Password:         creds.Password,
		DomainName:       creds.Domain,
		TenantName:       creds.Tenant,
		TenantID:         creds.TenantID,
	}

	client, err := goopenstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (os *Provider) getNetClient(cloud kubermaticv1.CloudSpec) (*gophercloud.ServiceClient, error) {
	authClient, err := os.getAuthClient(cloud)
	if err != nil {
		return nil, err
	}

	return goopenstack.NewNetworkV2(authClient, gophercloud.EndpointOpts{Region: os.dc.Region})
}

// GetSubnets list all available subnet ids fot a given CloudSpec
func (os *Provider) GetSubnets(cloud kubermaticv1.CloudSpec, networkID string) ([]ossubnets.Subnet, error) {
	serviceClient, err := os.getNetClient(cloud)
	if err != nil {
		return nil, fmt.Errorf("couldn't get auth client: %v", err)
	}

	subnets, err := getSubnetForNetwork(serviceClient, networkID)
	if err != nil {
		return nil, err
	}

	return subnets, nil
}

func (os *Provider) AddICMPRulesIfRequired(cluster *kubermaticv1.Cluster) error {
	if cluster.Spec.Cloud.Openstack.SecurityGroups == "" {
		return nil
	}
	sgName := cluster.Spec.Cloud.Openstack.SecurityGroups

	netClient, err := os.getNetClient(cluster.Spec.Cloud)
	if err != nil {
		return fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	// We can only get security groups by ID and can't be sure that whats on the cluster
	securityGroups, err := getSecurityGroups(netClient, ossecuritygroups.ListOpts{Name: sgName})
	if err != nil {
		return fmt.Errorf("failed to list security groups: %v", err)
	}

	for _, sg := range securityGroups {
		if err := addICMPRulesToSecurityGroupIfNecesary(cluster, sg, netClient); err != nil {
			return fmt.Errorf("failed to add rules for ICMP to security group %q: %v", sg.ID, err)
		}
	}
	return nil
}

func addICMPRulesToSecurityGroupIfNecesary(cluster *kubermaticv1.Cluster, secGroup ossecuritygroups.SecGroup, netClient *gophercloud.ServiceClient) error {
	var hasIPV4Rule, hasIPV6Rule bool
	for _, rule := range secGroup.Rules {
		if rule.Direction == string(osecuritygrouprules.DirIngress) {
			if rule.EtherType == string(osecuritygrouprules.EtherType4) && rule.Protocol == string(osecuritygrouprules.ProtocolICMP) {
				hasIPV4Rule = true
			}
			if rule.EtherType == string(osecuritygrouprules.EtherType6) && rule.Protocol == string(osecuritygrouprules.ProtocolIPv6ICMP) {
				hasIPV6Rule = true
			}
		}
	}

	var rulesToCreate []osecuritygrouprules.CreateOpts
	if !hasIPV4Rule {
		rulesToCreate = append(rulesToCreate, osecuritygrouprules.CreateOpts{
			Direction:  osecuritygrouprules.DirIngress,
			EtherType:  osecuritygrouprules.EtherType4,
			SecGroupID: secGroup.ID,
			Protocol:   osecuritygrouprules.ProtocolICMP,
		})
		glog.Infof("Adding ICMP allow rule to cluster %q", cluster.Name)
	}
	if !hasIPV6Rule {
		rulesToCreate = append(rulesToCreate, osecuritygrouprules.CreateOpts{
			Direction:  osecuritygrouprules.DirIngress,
			EtherType:  osecuritygrouprules.EtherType6,
			SecGroupID: secGroup.ID,
			Protocol:   osecuritygrouprules.ProtocolIPv6ICMP,
		})
		glog.Infof("Adding ICMP6 allow rule to cluster %q", cluster.Name)
	}

	for _, rule := range rulesToCreate {
		res := osecuritygrouprules.Create(netClient, rule)
		if res.Err != nil {
			return fmt.Errorf("failed to create security group rule: %v", res.Err)
		}
		if _, err := res.Extract(); err != nil {
			return fmt.Errorf("failed to extract result after security group creation: %v", err)
		}
	}

	return nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (os *Provider) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

type Credentials struct {
	Username string
	Password string
	Tenant   string
	TenantID string
	Domain   string
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (Credentials, error) {
	username := cloud.Openstack.Username
	password := cloud.Openstack.Password
	tenant := cloud.Openstack.Tenant
	tenantID := cloud.Openstack.TenantID
	domain := cloud.Openstack.Domain

	var err error

	if username == "" {
		if cloud.Openstack.CredentialsReference == nil {
			return Credentials{}, errors.New("no credentials provided")
		}
		username, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackUsername)
		if err != nil {
			return Credentials{}, err
		}
	}

	if password == "" {
		if cloud.Openstack.CredentialsReference == nil {
			return Credentials{}, errors.New("no credentials provided")
		}
		password, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackPassword)
		if err != nil {
			return Credentials{}, err
		}
	}

	if tenant == "" && cloud.Openstack.CredentialsReference != nil && cloud.Openstack.CredentialsReference.Name != "" {
		tenant, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackTenant)
		if err != nil {
			return Credentials{}, err
		}
	}

	if tenantID == "" && cloud.Openstack.CredentialsReference != nil && cloud.Openstack.CredentialsReference.Name != "" {
		tenantID, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackTenantID)
		if err != nil {
			return Credentials{}, err
		}
	}

	if domain == "" {
		if cloud.Openstack.CredentialsReference == nil {
			return Credentials{}, errors.New("no credentials provided")
		}
		domain, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackDomain)
		if err != nil {
			return Credentials{}, err
		}
	}

	return Credentials{
		Username: username,
		Password: password,
		Tenant:   tenant,
		TenantID: tenantID,
		Domain:   domain,
	}, nil
}
