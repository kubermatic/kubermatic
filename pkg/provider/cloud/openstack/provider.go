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

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/klog"
)

const (
	// SecurityGroupCleanupFinalizer will instruct the deletion of the security group.
	SecurityGroupCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-security-group"
	// OldNetworkCleanupFinalizer will instruct the deletion of all network components. Router, Network, Subnet
	// Deprecated: Got split into dedicated finalizers.
	OldNetworkCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-network"

	// NetworkCleanupFinalizer will instruct the deletion of the network.
	NetworkCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-network-v2"
	// SubnetCleanupFinalizer will instruct the deletion of the subnet.
	SubnetCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-subnet-v2"
	// RouterCleanupFinalizer will instruct the deletion of the router.
	RouterCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-router-v2"
	// RouterSubnetLinkCleanupFinalizer will instruct the deletion of the link between the router and the subnet.
	RouterSubnetLinkCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-router-subnet-link-v2"
)

type getClientFunc func(cluster kubermaticv1.CloudSpec, dc *kubermaticv1.DatacenterSpecOpenstack, secretKeySelector provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error)

// Provider is a struct that implements CloudProvider interface.
type Provider struct {
	dc                *kubermaticv1.DatacenterSpecOpenstack
	secretKeySelector provider.SecretKeySelectorValueFunc
	caBundle          *x509.CertPool
	getClientFunc     getClientFunc
}

// NewCloudProvider creates a new openstack provider.
func NewCloudProvider(
	dc *kubermaticv1.Datacenter,
	secretKeyGetter provider.SecretKeySelectorValueFunc,
	caBundle *x509.CertPool,
) (*Provider, error) {
	if dc.Spec.Openstack == nil {
		return nil, errors.New("datacenter is not an Openstack datacenter")
	}
	return &Provider{
		dc:                dc.Spec.Openstack,
		secretKeySelector: secretKeyGetter,
		caBundle:          caBundle,
		getClientFunc:     getNetClientForCluster,
	}, nil
}

var _ provider.CloudProvider = &Provider{}

// DefaultCloudSpec adds defaults to the cloud spec.
func (os *Provider) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

// ValidateCloudSpec validates the given CloudSpec.
func (os *Provider) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	netClient, err := os.getClientFunc(spec, os.dc, os.secretKeySelector, os.caBundle)
	if err != nil {
		return err
	}

	if spec.Openstack.SecurityGroups != "" {
		if err := validateSecurityGroupsExist(netClient, strings.Split(spec.Openstack.SecurityGroups, ",")); err != nil {
			return err
		}
	}

	if spec.Openstack.Network != "" {
		network, err := getNetworkByName(netClient, spec.Openstack.Network, false)
		if err != nil {
			return fmt.Errorf("failed to get network %q: %w", spec.Openstack.Network, err)
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
			return fmt.Errorf("failed to get floating ip pool %q: %w", spec.Openstack.FloatingIPPool, err)
		}
	}

	return nil
}

// validateExistingSubnetOverlap checks whether any subnets in the given network overlap with the default subnet CIDR.
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

// We start by adding the finalizers, note that this is safe because the
// clean-up is idempotent, if the cluster is deleted when resources associated
// to the finalizer are not created yet, it does not fail.
// The reason behind is that we have several controllers adding finalizers
// to the Cluster resource at the moment, and to avoid race conditions we
// need to use optimistic locking and return immediately in case of
// conflicts to retry later.
func ensureFinalizers(cluster *kubermaticv1.Cluster, finalizers []string, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var err error

	if len(finalizers) > 0 {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.AddFinalizer(cluster, finalizers...)
		})
		if err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

// InitializeCloudProvider initializes a cluster, in particular
// creates security group and network configuration.
func (os *Provider) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	netClient, err := os.getClientFunc(cluster.Spec.Cloud, os.dc, os.secretKeySelector, os.caBundle)
	if err != nil {
		return nil, err
	}

	var routerID string
	var finalizers []string
	// if security group has to be created add the corresponding finalizer.
	if cluster.Spec.Cloud.Openstack.SecurityGroups == "" {
		finalizers = append(finalizers, SecurityGroupCleanupFinalizer)
	}
	// If network has to be created add associated finalizer.
	if cluster.Spec.Cloud.Openstack.Network == "" {
		finalizers = append(finalizers, NetworkCleanupFinalizer)
	}
	// If subnet has to be created, router and router port should be
	// created too thus we add the finalizers.
	if cluster.Spec.Cloud.Openstack.SubnetID == "" {
		finalizers = append(finalizers, SubnetCleanupFinalizer, RouterCleanupFinalizer, RouterSubnetLinkCleanupFinalizer)
	} else if cluster.Spec.Cloud.Openstack.RouterID == "" {
		var err error
		routerID, err = getRouterIDForSubnet(netClient, cluster.Spec.Cloud.Openstack.SubnetID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify that the subnet '%s' has a router attached: %w", cluster.Spec.Cloud.Openstack.SubnetID, err)
		}
		// Subnet exists but we need to create the router
		if routerID == "" {
			finalizers = append(finalizers, RouterCleanupFinalizer, RouterSubnetLinkCleanupFinalizer)
		}
	}

	cluster, err = ensureFinalizers(cluster, finalizers, update)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure finalizers: %w", err)
	}

	if cluster.Spec.Cloud.Openstack.FloatingIPPool == "" {
		extNetwork, err := getExternalNetwork(netClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get external network: %w", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.FloatingIPPool = extNetwork.Name
			// We're just searching for the floating ip pool here & don't create anything. Thus no need to create a finalizer
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update cluster floating IP pool: %w", err)
		}
	}

	if cluster.Spec.Cloud.Openstack.SecurityGroups == "" {
		lowPort, highPort := resources.NewTemplateDataBuilder().
			WithNodePortRange(cluster.Spec.ComponentsOverride.Apiserver.NodePortRange).
			WithCluster(cluster).
			Build().
			NodePorts()

		nodePortsAllowedIPRange := cluster.Spec.Cloud.Openstack.NodePortsAllowedIPRange
		if nodePortsAllowedIPRange == "" {
			nodePortsAllowedIPRange = "0.0.0.0/0"
		}

		req := createKubermaticSecurityGroupRequest{
			clusterName:             cluster.Name,
			lowPort:                 lowPort,
			highPort:                highPort,
			nodePortsAllowedIPRange: nodePortsAllowedIPRange,
		}

		secGroupName, err := createKubermaticSecurityGroup(netClient, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic security group: %w", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.SecurityGroups = secGroupName
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add security group cleanup finalizer: %w", err)
		}
	}

	if cluster.Spec.Cloud.Openstack.Network == "" {
		network, err := createKubermaticNetwork(netClient, cluster.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic network: %w", err)
		}
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.Network = network.Name
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add network cleanup finalizer: %w", err)
		}
	}

	network, err := getNetworkByName(netClient, cluster.Spec.Cloud.Openstack.Network, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get network '%s': %w", cluster.Spec.Cloud.Openstack.Network, err)
	}

	if cluster.Spec.Cloud.Openstack.SubnetID == "" {
		subnet, err := createKubermaticSubnet(netClient, cluster.Name, network.ID, os.dc.DNSServers)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic subnet: %w", err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.SubnetID = subnet.ID
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add subnet cleanup finalizer: %w", err)
		}
	}

	if cluster.Spec.Cloud.Openstack.RouterID == "" {
		if routerID == "" {
			// No Router exists -> Create a router
			router, err := createKubermaticRouter(netClient, cluster.Name, cluster.Spec.Cloud.Openstack.FloatingIPPool)
			if err != nil {
				return nil, fmt.Errorf("failed to create the kubermatic router: %w", err)
			}
			cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
				cluster.Spec.Cloud.Openstack.RouterID = router.ID
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add router cleanup finalizer: %w", err)
			}
		} else {
			// A router already exists -> Reuse it but don't clean it up
			cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
				cluster.Spec.Cloud.Openstack.RouterID = routerID
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add router ID to cluster: %w", err)
			}
		}
	}

	// We only attach the router to the subnet if CloudProviderInfrastructure
	// health status is not up, meaning that there was no successful
	// reconciliations so far. This is to avoid hitting OpenStack API at each
	// iteration.
	// TODO: this is terrible, find a better way.
	if cluster.Status.ExtendedHealth.CloudProviderInfrastructure != kubermaticv1.HealthStatusUp &&
		kubernetes.HasFinalizer(cluster, RouterSubnetLinkCleanupFinalizer) {
		if _, err = attachSubnetToRouter(netClient, cluster.Spec.Cloud.Openstack.SubnetID, cluster.Spec.Cloud.Openstack.RouterID); err != nil {
			return nil, fmt.Errorf("failed to attach subnet to router: %w", err)
		}
	}

	return cluster, nil
}

// TODO: Hey, you! Yes, you! Why don't you implement reconciling for Openstack? Would be really cool :)
// func (os *Provider) ReconcileCluster(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
// 	return cluster, nil
// }

// CleanUpCloudProvider does the clean-up in particular:
// removes security group and network configuration.
func (os *Provider) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	netClient, err := os.getClientFunc(cluster.Spec.Cloud, os.dc, os.secretKeySelector, os.caBundle)
	if err != nil {
		return nil, err
	}

	if kubernetes.HasFinalizer(cluster, SecurityGroupCleanupFinalizer) {
		for _, g := range strings.Split(cluster.Spec.Cloud.Openstack.SecurityGroups, ",") {
			if err := deleteSecurityGroup(netClient, strings.TrimSpace(g)); err != nil {
				if !isNotFoundErr(err) {
					return nil, fmt.Errorf("failed to delete security group %q: %w", g, err)
				}
			}
		}
	}

	if kubernetes.HasFinalizer(cluster, RouterSubnetLinkCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if _, err = detachSubnetFromRouter(netClient, cluster.Spec.Cloud.Openstack.SubnetID, cluster.Spec.Cloud.Openstack.RouterID); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to detach subnet from router: %w", err)
			}
		}
	}

	if kubernetes.HasFinalizer(cluster, SubnetCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if err := deleteSubnet(netClient, cluster.Spec.Cloud.Openstack.SubnetID); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete subnet '%s': %w", cluster.Spec.Cloud.Openstack.SubnetID, err)
			}
		}
	}

	if kubernetes.HasFinalizer(cluster, NetworkCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if err = deleteNetworkByName(netClient, cluster.Spec.Cloud.Openstack.Network); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete network '%s': %w", cluster.Spec.Cloud.Openstack.Network, err)
			}
		}
	}

	if kubernetes.HasFinalizer(cluster, RouterCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if err = deleteRouter(netClient, cluster.Spec.Cloud.Openstack.RouterID); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete router '%s': %w", cluster.Spec.Cloud.Openstack.RouterID, err)
			}
		}
	}

	// Relying on the idempotence of the clean-up steps we remove all finalizers in
	// one shot only when the clean-up is completed.
	cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kubernetes.RemoveFinalizer(
			cluster,
			SecurityGroupCleanupFinalizer,
			RouterSubnetLinkCleanupFinalizer,
			SubnetCleanupFinalizer,
			NetworkCleanupFinalizer,
			RouterCleanupFinalizer,
			OldNetworkCleanupFinalizer,
		)
	})
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

// GetFlavors lists available flavors for the given CloudSpec.DatacenterName and OpenstackSpec.Region.
func GetFlavors(authURL, region string, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) ([]osflavors.Flavor, error) {
	authClient, err := getAuthClient(authURL, credentials, caBundle)
	if err != nil {
		return nil, err
	}
	flavors, err := getFlavors(authClient, region)
	if err != nil {
		return nil, err
	}

	return flavors, nil
}

// GetTenants lists all available tenents for the given CloudSpec.DatacenterName.
func GetTenants(authURL, region string, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) ([]osprojects.Project, error) {
	authClient, err := getAuthClient(authURL, credentials, caBundle)
	if err != nil {
		return nil, fmt.Errorf("couldn't get auth client: %w", err)
	}

	tenants, err := getTenants(authClient, region)
	if err != nil {
		return nil, fmt.Errorf("couldn't get tenants for region %s: %w", region, err)
	}

	return tenants, nil
}

// GetNetworks lists all available networks for the given CloudSpec.DatacenterName.
func GetNetworks(authURL, region string, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) ([]NetworkWithExternalExt, error) {
	authClient, err := getNetClient(authURL, region, credentials, caBundle)
	if err != nil {
		return nil, fmt.Errorf("couldn't get auth client: %w", err)
	}

	networks, err := getAllNetworks(authClient, osnetworks.ListOpts{})
	if err != nil {
		return nil, fmt.Errorf("couldn't get networks: %w", err)
	}

	return networks, nil
}

// GetSecurityGroups lists all available security groups for the given CloudSpec.DatacenterName.
func GetSecurityGroups(authURL, region string, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) ([]ossecuritygroups.SecGroup, error) {
	netClient, err := getNetClient(authURL, region, credentials, caBundle)
	if err != nil {
		return nil, fmt.Errorf("couldn't get auth client: %w", err)
	}

	page, err := ossecuritygroups.List(netClient, ossecuritygroups.ListOpts{}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list security groups: %w", err)
	}
	secGroups, err := ossecuritygroups.ExtractGroups(page)
	if err != nil {
		return nil, fmt.Errorf("failed to extract security groups: %w", err)
	}
	return secGroups, nil
}

// GetAvailabilityZones lists availability zones for the given CloudSpec.DatacenterName and OpenstackSpec.Region.
func GetAvailabilityZones(authURL, region string, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) ([]osavailabilityzones.AvailabilityZone, error) {
	computeClient, err := getComputeClient(authURL, region, credentials, caBundle)
	if err != nil {
		return nil, err
	}
	availabilityZones, err := getAvailabilityZones(computeClient)
	if err != nil {
		return nil, err
	}

	return availabilityZones, nil
}

func getAuthClient(authURL string, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) (*gophercloud.ProviderClient, error) {
	opts := gophercloud.AuthOptions{
		IdentityEndpoint:            authURL,
		Username:                    credentials.Username,
		Password:                    credentials.Password,
		DomainName:                  credentials.Domain,
		TenantName:                  credentials.Project,
		TenantID:                    credentials.ProjectID,
		ApplicationCredentialID:     credentials.ApplicationCredentialID,
		ApplicationCredentialSecret: credentials.ApplicationCredentialSecret,
		TokenID:                     credentials.Token,
	}

	client, err := goopenstack.NewClient(authURL)
	if err != nil {
		return nil, err
	}

	if client != nil {
		// overwrite the default host/root CA Bundle with the proper CA Bundle
		client.HTTPClient.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: caBundle}}
	}

	err = goopenstack.Authenticate(client, opts)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getNetClient(authURL, region string, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
	authClient, err := getAuthClient(authURL, credentials, caBundle)
	if err != nil {
		return nil, err
	}

	serviceClient, err := goopenstack.NewNetworkV2(authClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		// this is special case for services that span only one region.
		if isEndpointNotFoundErr(err) {
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

func getComputeClient(authURL, region string, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
	authClient, err := getAuthClient(authURL, credentials, caBundle)
	if err != nil {
		return nil, err
	}

	serviceClient, err := goopenstack.NewComputeV2(authClient, gophercloud.EndpointOpts{Region: region})
	if err != nil {
		// this is special case for services that span only one region.
		if isEndpointNotFoundErr(err) {
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

// GetSubnets list all available subnet ids for a given CloudSpec.
func GetSubnets(authURL, region, networkID string, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) ([]ossubnets.Subnet, error) {
	serviceClient, err := getNetClient(authURL, region, credentials, caBundle)
	if err != nil {
		return nil, fmt.Errorf("couldn't get auth client: %w", err)
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

	netClient, err := getNetClient(os.dc.AuthURL, os.dc.Region, creds, os.caBundle)
	if err != nil {
		return fmt.Errorf("failed to create a authenticated openstack client: %w", err)
	}

	// We can only get security groups by ID and can't be sure that what's on the cluster
	securityGroups, err := getSecurityGroups(netClient, ossecuritygroups.ListOpts{Name: sgName})
	if err != nil {
		return fmt.Errorf("failed to list security groups: %w", err)
	}

	for _, sg := range securityGroups {
		if err := addICMPRulesToSecurityGroupIfNecesary(cluster, sg, netClient); err != nil {
			return fmt.Errorf("failed to add rules for ICMP to security group %q: %w", sg.ID, err)
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
			return fmt.Errorf("failed to create security group rule: %w", res.Err)
		}
		if _, err := res.Extract(); err != nil {
			return fmt.Errorf("failed to extract result after security group creation: %w", err)
		}
	}

	return nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (os *Provider) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

func getNetClientForCluster(cluster kubermaticv1.CloudSpec, dc *kubermaticv1.DatacenterSpecOpenstack, secretKeySelector provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
	creds, err := GetCredentialsForCluster(cluster, secretKeySelector)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	netClient, err := getNetClient(dc.AuthURL, dc.Region, creds, caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create a authenticated openstack client: %w", err)
	}
	return netClient, nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
// The user can choose three ways for authentication. The first is a token. Second through Application Credentials.
// The last one uses a username and password. Those methods work exclusively.
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (*resources.OpenstackCredentials, error) {
	username := cloud.Openstack.Username
	password := cloud.Openstack.Password
	project := cloud.Openstack.Project
	projectID := cloud.Openstack.ProjectID
	domain := cloud.Openstack.Domain
	applicationCredentialID := cloud.Openstack.ApplicationCredentialID
	applicationCredentialSecret := cloud.Openstack.ApplicationCredentialSecret
	useToken := cloud.Openstack.UseToken
	token := cloud.Openstack.Token
	var err error

	if domain == "" {
		if cloud.Openstack.CredentialsReference == nil {
			return &resources.OpenstackCredentials{}, errors.New("no credentials provided")
		}
		domain, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackDomain)
		if err != nil {
			return &resources.OpenstackCredentials{}, err
		}
	}

	if useToken && token != "" {
		return &resources.OpenstackCredentials{
			Token:  token,
			Domain: domain,
		}, nil
	}

	if !useToken && cloud.Openstack.CredentialsReference != nil {
		token, _ := secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackToken)
		if token != "" {
			return &resources.OpenstackCredentials{
				Token:  token,
				Domain: domain,
			}, nil
		}
	}

	if applicationCredentialID != "" && applicationCredentialSecret != "" {
		return &resources.OpenstackCredentials{
			ApplicationCredentialSecret: applicationCredentialSecret,
			ApplicationCredentialID:     applicationCredentialID,
			Domain:                      domain,
		}, nil
	}

	if applicationCredentialID == "" && cloud.Openstack.CredentialsReference != nil {
		applicationCredentialID, _ = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackApplicationCredentialID)
		if applicationCredentialID != "" {
			applicationCredentialSecret, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackApplicationCredentialSecret)
			if err != nil {
				return &resources.OpenstackCredentials{}, err
			}

			return &resources.OpenstackCredentials{
				ApplicationCredentialSecret: applicationCredentialSecret,
				ApplicationCredentialID:     applicationCredentialID,
				Domain:                      domain,
			}, nil
		}
	}

	if username == "" {
		if cloud.Openstack.CredentialsReference == nil {
			return &resources.OpenstackCredentials{}, errors.New("no credentials provided")
		}
		username, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackUsername)
		if err != nil {
			return &resources.OpenstackCredentials{}, err
		}
	}

	if password == "" {
		if cloud.Openstack.CredentialsReference == nil {
			return &resources.OpenstackCredentials{}, errors.New("no credentials provided")
		}
		password, err = secretKeySelector(cloud.Openstack.CredentialsReference, resources.OpenstackPassword)
		if err != nil {
			return &resources.OpenstackCredentials{}, err
		}
	}

	if project == "" && cloud.Openstack.CredentialsReference != nil && cloud.Openstack.CredentialsReference.Name != "" {
		if project, err = firstKey(secretKeySelector, cloud.Openstack.CredentialsReference, resources.OpenstackProject, resources.OpenstackTenant); err != nil {
			return &resources.OpenstackCredentials{}, err
		}
	}

	if projectID == "" && cloud.Openstack.CredentialsReference != nil && cloud.Openstack.CredentialsReference.Name != "" {
		if projectID, err = firstKey(secretKeySelector, cloud.Openstack.CredentialsReference, resources.OpenstackProjectID, resources.OpenstackTenantID); err != nil {
			return &resources.OpenstackCredentials{}, err
		}
	}

	return &resources.OpenstackCredentials{
		Username:                    username,
		Password:                    password,
		Project:                     project,
		ProjectID:                   projectID,
		Domain:                      domain,
		ApplicationCredentialID:     applicationCredentialID,
		ApplicationCredentialSecret: applicationCredentialSecret,
	}, nil
}

// firstKey read the secret and return value for the firstkey. if the firstKey does not exist, tries with
// fallbackKey. if the fallbackKey does not exist then return an error.
func firstKey(secretKeySelector provider.SecretKeySelectorValueFunc, configVar *providerconfig.GlobalSecretKeySelector, firstKey string, fallbackKey string) (string, error) {
	var value string
	var err error
	if value, err = secretKeySelector(configVar, firstKey); err != nil {
		// fallback
		if value, err = secretKeySelector(configVar, fallbackKey); err != nil {
			return "", err
		}
	}
	return value, nil
}

func ignoreRouterAlreadyHasPortInSubnetError(err error, subnetID string) error {
	matchString := fmt.Sprintf("Router already has a port on subnet %s", subnetID)

	var gopherCloud400Err gophercloud.ErrDefault400
	if !errors.As(err, &gopherCloud400Err) || !strings.Contains(string(gopherCloud400Err.Body), matchString) {
		return err
	}

	return nil
}

func ValidateCredentials(authURL, region string, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) error {
	computeClient, err := getComputeClient(authURL, region, credentials, caBundle)
	if err != nil {
		return err
	}
	_, err = getAvailabilityZones(computeClient)
	if err != nil {
		return err
	}
	return nil
}
