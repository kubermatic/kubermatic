/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package openstack

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gophercloud/gophercloud"
	goopenstack "github.com/gophercloud/gophercloud/openstack"
	osavailabilityzones "github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	osflavors "github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	osprojects "github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	ossecuritygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	osecuritygrouprules "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	osnetworks "github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	ossubnets "github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/klog"
)

const (
	// SecurityGroupCleanupFinalizer will instruct the deletion of the security group
	SecurityGroupCleanupFinalizer = "kubermatic.io/cleanup-openstack-security-group"
	// OldNetworkCleanupFinalizer will instruct the deletion of all network components. Router, Network, Subnet
	// Deprecated: Got split into dedicated finalizers
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
	caBundle          *x509.CertPool
	nodePortsRange    string
}

// NewCloudProvider creates a new openstack provider.
func NewCloudProvider(
	dc *kubermaticv1.Datacenter,
	secretKeyGetter provider.SecretKeySelectorValueFunc,
	caBundle *x509.CertPool,
	nodePortsRange string,
) (*Provider, error) {
	if dc.Spec.Openstack == nil {
		return nil, errors.New("datacenter is not an Openstack datacenter")
	}
	return &Provider{
		dc:                dc.Spec.Openstack,
		secretKeySelector: secretKeyGetter,
		caBundle:          caBundle,
		nodePortsRange:    nodePortsRange,
	}, nil
}

// DefaultCloudSpec adds defaults to the cloud spec
func (os *Provider) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

// ValidateCloudSpec validates the given CloudSpec
func (os *Provider) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	creds, err := GetCredentialsForCluster(spec, os.secretKeySelector)
	if err != nil {
		return err
	}

	netClient, err := getNetClient(creds.Username, creds.Password, creds.Domain, creds.Tenant, creds.TenantID, os.dc.AuthURL, os.dc.Region, os.caBundle)
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
	creds, err := GetCredentialsForCluster(cluster.Spec.Cloud, os.secretKeySelector)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %v", err)
	}

	netClient, err := getNetClient(creds.Username, creds.Password, creds.Domain, creds.Tenant, creds.TenantID, os.dc.AuthURL, os.dc.Region, os.caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if cluster.Spec.Cloud.Openstack.FloatingIPPool == "" {
		extNetwork, err := getExternalNetwork(netClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get external network: %v", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.FloatingIPPool = extNetwork.Name
			// We're just searching for the floating ip pool here & don't create anything. Thus no need to create a finalizer
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update cluster floating IP pool: %v", err)
		}
	}

	if cluster.Spec.Cloud.Openstack.SecurityGroups == "" {
		lowPort, highPort := resources.NewTemplateDataBuilder().
			WithNodePortRange(os.nodePortsRange).
			WithCluster(cluster).
			Build().
			NodePorts()

		req := createKubermaticSecurityGroupRequest{
			clusterName: cluster.Name,
			lowPort:     lowPort,
			highPort:    highPort,
		}

		secGroupName, err := createKubermaticSecurityGroup(netClient, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic security group: %v", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.SecurityGroups = secGroupName
			kubernetes.AddFinalizer(cluster, SecurityGroupCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add security group cleanup finalizer: %v", err)
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
			return nil, fmt.Errorf("failed to add network cleanup finalizer: %v", err)
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
			return nil, fmt.Errorf("failed to add subnet cleanup finalizer: %v", err)
		}
	}

	if cluster.Spec.Cloud.Openstack.RouterID == "" {
		// Check if the subnet has already a router
		routerID, err := getRouterIDForSubnet(netClient, cluster.Spec.Cloud.Openstack.SubnetID, network.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify that the subnet '%s' has a router attached: %v", cluster.Spec.Cloud.Openstack.SubnetID, err)
		}

		if routerID == "" {
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
				return nil, fmt.Errorf("failed to add router cleanup finalizer: %v", err)
			}
		} else {
			// A router already exists -> Reuse it but don't clean it up
			cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
				cluster.Spec.Cloud.Openstack.RouterID = routerID
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add router ID to cluster: %v", err)
			}
		}
	}

	// If we created the router, but have not created the router-subnet-link finalizer, we need to attach the subnet to the router
	// Otherwise the vm's won't have connectivity
	if kubernetes.HasFinalizer(cluster, RouterCleanupFinalizer) && !kubernetes.HasFinalizer(cluster, RouterSubnetLinkCleanupFinalizer) {
		if _, err = attachSubnetToRouter(netClient, cluster.Spec.Cloud.Openstack.SubnetID, cluster.Spec.Cloud.Openstack.RouterID); err != nil {
			return nil, fmt.Errorf("failed to attach subnet to router: %v", err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.AddFinalizer(cluster, RouterSubnetLinkCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add router subnet cleanup finalizer: %v", err)
		}
	}

	return cluster, nil
}

// CleanUpCloudProvider does the clean-up in particular:
// removes security group and network configuration
func (os *Provider) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	creds, err := GetCredentialsForCluster(cluster.Spec.Cloud, os.secretKeySelector)
	if err != nil {
		return nil, err
	}

	netClient, err := getNetClient(creds.Username, creds.Password, creds.Domain, creds.Tenant, creds.TenantID, os.dc.AuthURL, os.dc.Region, os.caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	if kubernetes.HasFinalizer(cluster, SecurityGroupCleanupFinalizer) {
		for _, g := range strings.Split(cluster.Spec.Cloud.Openstack.SecurityGroups, ",") {
			if err := deleteSecurityGroup(netClient, strings.TrimSpace(g)); err != nil {
				if !isNotFoundErr(err) {
					return nil, fmt.Errorf("failed to delete security group %q: %v", g, err)
				}
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
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete subnet '%s': %v", cluster.Spec.Cloud.Openstack.SubnetID, err)
			}
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
func GetFlavors(username, password, domain, tenant, tenantID, authURL, region string, caBundle *x509.CertPool) ([]osflavors.Flavor, error) {
	authClient, err := getAuthClient(username, password, domain, tenant, tenantID, authURL, caBundle)
	if err != nil {
		return nil, err
	}
	flavors, err := getFlavors(authClient, region)
	if err != nil {
		return nil, err
	}

	return flavors, nil
}

// GetTenants lists all available tenents for the given CloudSpec.DatacenterName
func GetTenants(username, password, domain, tenant, tenantID, authURL, region string, caBundle *x509.CertPool) ([]osprojects.Project, error) {
	authClient, err := getAuthClient(username, password, domain, tenant, tenantID, authURL, caBundle)
	if err != nil {
		return nil, fmt.Errorf("couldn't get auth client: %v", err)
	}

	tenants, err := getTenants(authClient, region)
	if err != nil {
		return nil, fmt.Errorf("couldn't get tenants for region %s: %v", region, err)
	}

	return tenants, nil
}

// GetNetworks lists all available networks for the given CloudSpec.DatacenterName
func GetNetworks(username, password, domain, tenant, tenantID, authURL, region string, caBundle *x509.CertPool) ([]NetworkWithExternalExt, error) {
	authClient, err := getNetClient(username, password, domain, tenant, tenantID, authURL, region, caBundle)
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
func GetSecurityGroups(username, password, domain, tenant, tenantID, authURL, region string, caBundle *x509.CertPool) ([]ossecuritygroups.SecGroup, error) {
	netClient, err := getNetClient(username, password, domain, tenant, tenantID, authURL, region, caBundle)
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

// GetAvailabilityZones lists availability zones for the given CloudSpec.DatacenterName and OpenstackSpec.Region
func GetAvailabilityZones(username, password, domain, tenant, tenantID, authURL, region string, caBundle *x509.CertPool) ([]osavailabilityzones.AvailabilityZone, error) {
	computeClient, err := getComputeClient(username, password, domain, tenant, tenantID, authURL, region, caBundle)
	if err != nil {
		return nil, err
	}
	availabilityZones, err := getAvailabilityZones(computeClient)
	if err != nil {
		return nil, err
	}

	return availabilityZones, nil
}

func getAuthClient(username, password, domain, tenant, tenantID, authURL string, caBundle *x509.CertPool) (*gophercloud.ProviderClient, error) {
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: authURL,
		Username:         username,
		Password:         password,
		DomainName:       domain,
		TenantName:       tenant,
		TenantID:         tenantID,
	}

	client, err := goopenstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, err
	}
	if client != nil {
		// overwrite the default host/root CA Bundle with the proper CA Bundle
		client.HTTPClient.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: caBundle}}
	}

	return client, nil
}

func getNetClient(username, password, domain, tenant, tenantID, authURL, region string, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
	authClient, err := getAuthClient(username, password, domain, tenant, tenantID, authURL, caBundle)
	if err != nil {
		return nil, err
	}

	serviceClient, err := goopenstack.NewNetworkV2(authClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		// this is special case for  services that span only one region.
		//nolint:gosimple
		//lint:ignore S1020 false positive, we must do the errcheck regardless of if its an ErrEndpointNotFound
		if _, ok := err.(*gophercloud.ErrEndpointNotFound); ok {
			serviceClient, err = goopenstack.NewNetworkV2(authClient, gophercloud.EndpointOpts{})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return serviceClient, err
}

func getComputeClient(username, password, domain, tenant, tenantID, authURL, region string, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
	authClient, err := getAuthClient(username, password, domain, tenant, tenantID, authURL, caBundle)
	if err != nil {
		return nil, err
	}

	serviceClient, err := goopenstack.NewComputeV2(authClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		// this is special case for  services that span only one region.
		//nolint:gosimple
		//lint:ignore S1020 false positive, we must do the errcheck regardless of if its an ErrEndpointNotFound
		if _, ok := err.(*gophercloud.ErrEndpointNotFound); ok {
			serviceClient, err = goopenstack.NewComputeV2(authClient, gophercloud.EndpointOpts{})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return serviceClient, err
}

// GetSubnets list all available subnet ids for a given CloudSpec
func GetSubnets(username, password, domain, tenant, tenantID, networkID, authURL, region string, caBundle *x509.CertPool) ([]ossubnets.Subnet, error) {
	serviceClient, err := getNetClient(username, password, domain, tenant, tenantID, authURL, region, caBundle)
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

	creds, err := GetCredentialsForCluster(cluster.Spec.Cloud, os.secretKeySelector)
	if err != nil {
		return err
	}

	netClient, err := getNetClient(creds.Username, creds.Password, creds.Domain, creds.Tenant, creds.TenantID, os.dc.AuthURL, os.dc.Region, os.caBundle)
	if err != nil {
		return fmt.Errorf("failed to create a authenticated openstack client: %v", err)
	}

	// We can only get security groups by ID and can't be sure that what's on the cluster
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
		klog.Infof("Adding ICMP allow rule to cluster %q", cluster.Name)
	}
	if !hasIPV6Rule {
		rulesToCreate = append(rulesToCreate, osecuritygrouprules.CreateOpts{
			Direction:  osecuritygrouprules.DirIngress,
			EtherType:  osecuritygrouprules.EtherType6,
			SecGroupID: secGroup.ID,
			Protocol:   osecuritygrouprules.ProtocolIPv6ICMP,
		})
		klog.Infof("Adding ICMP6 allow rule to cluster %q", cluster.Name)
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

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (resources.OpenstackCredentials, error) {
	username := cloud.Openstack.Username
	password := cloud.Openstack.Password
	tenant := cloud.Openstack.Tenant
	tenantID := cloud.Openstack.TenantID
	domain := cloud.Openstack.Domain

	var err error

	if username == "" {
		if cloud.Openstack.CredentialsReference == nil {
			return resources.OpenstackCredentials{}, errors.New("no credentials provided")
		}
		username, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackUsername)
		if err != nil {
			return resources.OpenstackCredentials{}, err
		}
	}

	if password == "" {
		if cloud.Openstack.CredentialsReference == nil {
			return resources.OpenstackCredentials{}, errors.New("no credentials provided")
		}
		password, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackPassword)
		if err != nil {
			return resources.OpenstackCredentials{}, err
		}
	}

	if tenant == "" && cloud.Openstack.CredentialsReference != nil && cloud.Openstack.CredentialsReference.Name != "" {
		tenant, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackTenant)
		if err != nil {
			return resources.OpenstackCredentials{}, err
		}
	}

	if tenantID == "" && cloud.Openstack.CredentialsReference != nil && cloud.Openstack.CredentialsReference.Name != "" {
		tenantID, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackTenantID)
		if err != nil {
			return resources.OpenstackCredentials{}, err
		}
	}

	if domain == "" {
		if cloud.Openstack.CredentialsReference == nil {
			return resources.OpenstackCredentials{}, errors.New("no credentials provided")
		}
		domain, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackDomain)
		if err != nil {
			return resources.OpenstackCredentials{}, err
		}
	}

	return resources.OpenstackCredentials{
		Username: username,
		Password: password,
		Tenant:   tenant,
		TenantID: tenantID,
		Domain:   domain,
	}, nil
}

func ignoreRouterAlreadyHasPortInSubnetError(err error, subnetID string) error {
	gopherCloud400Err, ok := err.(gophercloud.ErrDefault400)
	if !ok {
		return err
	}

	matchString := fmt.Sprintf("Router already has a port on subnet %s", subnetID)
	if !strings.Contains(string(gopherCloud400Err.Body), matchString) {
		return err
	}

	return nil
}
