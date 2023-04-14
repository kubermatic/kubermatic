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
	"context"
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
	ossubnetpools "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/subnetpools"
	osnetworks "github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	ossubnets "github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/provider"
	"k8c.io/kubermatic/v3/pkg/resources"
)

const (
	// SecurityGroupCleanupFinalizer will instruct the deletion of the security group.
	SecurityGroupCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-security-group"
	// OldNetworkCleanupFinalizer will instruct the deletion of all network components. Router, Network, Subnet
	// Deprecated: Got split into dedicated finalizers.
	OldNetworkCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-network"

	// NetworkCleanupFinalizer will instruct the deletion of the network.
	NetworkCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-network-v2"
	// SubnetCleanupFinalizer will instruct the deletion of the IPv4 subnet.
	SubnetCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-subnet-v2"
	// IPv6SubnetCleanupFinalizer will instruct the deletion of the IPv6 subnet.
	IPv6SubnetCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-subnet-ipv6"
	// RouterCleanupFinalizer will instruct the deletion of the router.
	RouterCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-router-v2"
	// RouterSubnetLinkCleanupFinalizer will instruct the deletion of the link between the router and the IPv4 subnet.
	RouterSubnetLinkCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-router-subnet-link-v2"
	// RouterIPv6SubnetLinkCleanupFinalizer will instruct the deletion of the link between the router and the IPv6 subnet.
	RouterIPv6SubnetLinkCleanupFinalizer = "kubermatic.k8c.io/cleanup-openstack-router-subnet-link-ipv6"
)

type getClientFunc func(ctx context.Context, cluster kubermaticv1.CloudSpec, dc *kubermaticv1.DatacenterSpecOpenStack, secretKeySelector provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error)

// Provider is a struct that implements CloudProvider interface.
type Provider struct {
	dc                *kubermaticv1.DatacenterSpecOpenStack
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
	if dc.Spec.Provider.OpenStack == nil {
		return nil, errors.New("datacenter is not an Openstack datacenter")
	}
	return &Provider{
		dc:                dc.Spec.Provider.OpenStack,
		secretKeySelector: secretKeyGetter,
		caBundle:          caBundle,
		getClientFunc:     getNetClientForCluster,
	}, nil
}

var _ provider.CloudProvider = &Provider{}

// DefaultCloudSpec adds defaults to the cloud spec.
func (os *Provider) DefaultCloudSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec) error {
	if spec.Cloud.OpenStack == nil {
		return errors.New("no Openstack cloud spec found")
	}
	switch spec.ClusterNetwork.IPFamily {
	case kubermaticv1.IPFamilyIPv4:
		spec.Cloud.OpenStack.NodePortsAllowedIPRanges = &kubermaticv1.NetworkRanges{
			CIDRBlocks: []string{resources.IPv4MatchAnyCIDR},
		}
	case kubermaticv1.IPFamilyDualStack:
		spec.Cloud.OpenStack.NodePortsAllowedIPRanges = &kubermaticv1.NetworkRanges{
			CIDRBlocks: []string{resources.IPv4MatchAnyCIDR, resources.IPv6MatchAnyCIDR},
		}
	}
	return nil
}

// ValidateCloudSpec validates the given CloudSpec.
func (os *Provider) ValidateCloudSpec(ctx context.Context, spec kubermaticv1.CloudSpec) error {
	netClient, err := os.getClientFunc(ctx, spec, os.dc, os.secretKeySelector, os.caBundle)
	if err != nil {
		return err
	}

	if spec.OpenStack.SecurityGroups != "" {
		if err := validateSecurityGroupsExist(netClient, strings.Split(spec.OpenStack.SecurityGroups, ",")); err != nil {
			return err
		}
	}

	if spec.OpenStack.Network != "" {
		network, err := getNetworkByName(netClient, spec.OpenStack.Network, false)
		if err != nil {
			return fmt.Errorf("failed to get network %q: %w", spec.OpenStack.Network, err)
		}

		// If we're going to create a subnet in an existing network,
		// let's check whether any existing subnets collide with our range.
		if spec.OpenStack.SubnetID == "" {
			if err = validateExistingSubnetOverlap(network.ID, netClient); err != nil {
				return err
			}
		}
	}

	if spec.OpenStack.FloatingIPPool != "" {
		_, err := getNetworkByName(netClient, spec.OpenStack.FloatingIPPool, true)
		if err != nil {
			return fmt.Errorf("failed to get floating ip pool %q: %w", spec.OpenStack.FloatingIPPool, err)
		}
	}

	if spec.OpenStack.IPv6SubnetPool != "" {
		subnetPool, err := getSubnetPoolByName(netClient, spec.OpenStack.IPv6SubnetPool)
		if err != nil {
			return fmt.Errorf("failed to get subnet pool %q: %w", spec.OpenStack.IPv6SubnetPool, err)
		}
		if subnetPool.IPversion != 6 {
			return fmt.Errorf("provided IPv6 subnet pool %q has incorrect IP version: %d", spec.OpenStack.IPv6SubnetPool, subnetPool.IPversion)
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
func ensureFinalizers(ctx context.Context, cluster *kubermaticv1.Cluster, finalizers []string, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var err error

	if len(finalizers) > 0 {
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
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
func (os *Provider) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	netClient, err := os.getClientFunc(ctx, cluster.Spec.Cloud, os.dc, os.secretKeySelector, os.caBundle)
	if err != nil {
		return nil, err
	}

	var routerID string
	var finalizers []string

	ipv4Network := cluster.Spec.ClusterNetwork.IsIPv4Only() || cluster.Spec.ClusterNetwork.IsDualStack()
	ipv6Network := cluster.Spec.ClusterNetwork.IsIPv6Only() || cluster.Spec.ClusterNetwork.IsDualStack()

	// if security group has to be created add the corresponding finalizer.
	if cluster.Spec.Cloud.OpenStack.SecurityGroups == "" {
		finalizers = append(finalizers, SecurityGroupCleanupFinalizer)
	}
	// If network has to be created add associated finalizer.
	if cluster.Spec.Cloud.OpenStack.Network == "" {
		finalizers = append(finalizers, NetworkCleanupFinalizer)
	}

	// if SubnetID is provided but RouterID not, try to retrieve RouterID
	if cluster.Spec.Cloud.OpenStack.SubnetID != "" && cluster.Spec.Cloud.OpenStack.RouterID == "" {
		var err error
		routerID, err = getRouterIDForSubnet(netClient, cluster.Spec.Cloud.OpenStack.SubnetID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify that the subnet '%s' has a router attached: %w", cluster.Spec.Cloud.OpenStack.SubnetID, err)
		}
	}
	if cluster.Spec.Cloud.OpenStack.IPv6SubnetID != "" && cluster.Spec.Cloud.OpenStack.RouterID == "" && routerID == "" {
		var err error
		routerID, err = getRouterIDForSubnet(netClient, cluster.Spec.Cloud.OpenStack.IPv6SubnetID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify that the subnet '%s' has a router attached: %w", cluster.Spec.Cloud.OpenStack.IPv6SubnetID, err)
		}
	}
	// If router has to be created, add associated finalizer.
	if cluster.Spec.Cloud.OpenStack.RouterID == "" && routerID == "" {
		finalizers = append(finalizers, RouterCleanupFinalizer)
	}

	if ipv4Network {
		// If subnet has to be created, add associated finalizer.
		if cluster.Spec.Cloud.OpenStack.SubnetID == "" {
			finalizers = append(finalizers, SubnetCleanupFinalizer)
		}
		// If subnet or router has to be created, subnet needs to be attached to the router.
		if cluster.Spec.Cloud.OpenStack.SubnetID == "" || (cluster.Spec.Cloud.OpenStack.RouterID == "" && routerID == "") {
			finalizers = append(finalizers, RouterSubnetLinkCleanupFinalizer)
		}
	}
	if ipv6Network {
		// If subnet has to be created, add associated finalizer.
		if cluster.Spec.Cloud.OpenStack.IPv6SubnetID == "" {
			finalizers = append(finalizers, IPv6SubnetCleanupFinalizer)
		}
		// If subnet or router has to be created, subnet needs to be attached to the router.
		if cluster.Spec.Cloud.OpenStack.IPv6SubnetID == "" || (cluster.Spec.Cloud.OpenStack.RouterID == "" && routerID == "") {
			finalizers = append(finalizers, RouterIPv6SubnetLinkCleanupFinalizer)
		}
	}

	cluster, err = ensureFinalizers(ctx, cluster, finalizers, update)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure finalizers: %w", err)
	}

	if cluster.Spec.Cloud.OpenStack.FloatingIPPool == "" {
		extNetwork, err := getExternalNetwork(netClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get external network: %w", err)
		}
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.OpenStack.FloatingIPPool = extNetwork.Name
			// We're just searching for the floating ip pool here & don't create anything. Thus no need to create a finalizer
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update cluster floating IP pool: %w", err)
		}
	}

	if cluster.Spec.Cloud.OpenStack.SecurityGroups == "" {
		lowPort, highPort := resources.NewTemplateDataBuilder().
			WithNodePortRange(cluster.Spec.ComponentsOverride.Apiserver.NodePortRange).
			WithCluster(cluster).
			Build().
			NodePorts()

		req := createKubermaticSecurityGroupRequest{
			clusterName: cluster.Name,
			ipv4Rules:   ipv4Network,
			ipv6Rules:   ipv6Network,
			lowPort:     lowPort,
			highPort:    highPort,
		}

		req.nodePortsCIDRs = resources.GetNodePortsAllowedIPRanges(cluster, cluster.Spec.Cloud.OpenStack.NodePortsAllowedIPRanges, cluster.Spec.Cloud.OpenStack.NodePortsAllowedIPRange)

		secGroupName, err := createKubermaticSecurityGroup(netClient, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic security group: %w", err)
		}
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.OpenStack.SecurityGroups = secGroupName
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add security group cleanup finalizer: %w", err)
		}
	}

	if cluster.Spec.Cloud.OpenStack.Network == "" {
		network, err := createKubermaticNetwork(netClient, cluster.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic network: %w", err)
		}
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.OpenStack.Network = network.Name
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add network cleanup finalizer: %w", err)
		}
	}

	network, err := getNetworkByName(netClient, cluster.Spec.Cloud.OpenStack.Network, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get network '%s': %w", cluster.Spec.Cloud.OpenStack.Network, err)
	}

	if ipv4Network && cluster.Spec.Cloud.OpenStack.SubnetID == "" {
		subnet, err := createKubermaticSubnet(netClient, cluster.Name, network.ID, os.dc.DNSServers)
		if err != nil {
			return nil, fmt.Errorf("failed to create the kubermatic subnet: %w", err)
		}
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.OpenStack.SubnetID = subnet.ID
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add subnet cleanup finalizer: %w", err)
		}
	}

	if ipv6Network && cluster.Spec.Cloud.OpenStack.IPv6SubnetID == "" {
		subnet, err := createKubermaticIPv6Subnet(netClient, cluster.Name, network.ID, cluster.Spec.Cloud.OpenStack.IPv6SubnetPool, os.dc.DNSServers)
		if err != nil {
			return nil, fmt.Errorf("failed to create the v6 subnet: %w", err)
		}
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.OpenStack.IPv6SubnetID = subnet.ID
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update v6 subnet ID: %w", err)
		}
	}

	if cluster.Spec.Cloud.OpenStack.RouterID == "" {
		if routerID == "" {
			// No Router exists -> Create a router
			router, err := createKubermaticRouter(netClient, cluster.Name, cluster.Spec.Cloud.OpenStack.FloatingIPPool)
			if err != nil {
				return nil, fmt.Errorf("failed to create the kubermatic router: %w", err)
			}
			cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
				cluster.Spec.Cloud.OpenStack.RouterID = router.ID
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add router cleanup finalizer: %w", err)
			}
		} else {
			// A router already exists -> Reuse it but don't clean it up
			cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
				cluster.Spec.Cloud.OpenStack.RouterID = routerID
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
	if cluster.Status.ExtendedHealth.CloudProviderInfrastructure != kubermaticv1.HealthStatusUp {
		if kubernetes.HasFinalizer(cluster, RouterSubnetLinkCleanupFinalizer) {
			if _, err = attachSubnetToRouter(netClient, cluster.Spec.Cloud.OpenStack.SubnetID, cluster.Spec.Cloud.OpenStack.RouterID); err != nil {
				return nil, fmt.Errorf("failed to attach subnet to router: %w", err)
			}
		}
		if kubernetes.HasFinalizer(cluster, RouterIPv6SubnetLinkCleanupFinalizer) {
			if _, err = attachSubnetToRouter(netClient, cluster.Spec.Cloud.OpenStack.IPv6SubnetID, cluster.Spec.Cloud.OpenStack.RouterID); err != nil {
				return nil, fmt.Errorf("failed to attach subnet to router: %w", err)
			}
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
func (os *Provider) CleanUpCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	netClient, err := os.getClientFunc(ctx, cluster.Spec.Cloud, os.dc, os.secretKeySelector, os.caBundle)
	if err != nil {
		return nil, err
	}

	if kubernetes.HasFinalizer(cluster, SecurityGroupCleanupFinalizer) {
		for _, g := range strings.Split(cluster.Spec.Cloud.OpenStack.SecurityGroups, ",") {
			if err := deleteSecurityGroup(netClient, strings.TrimSpace(g)); err != nil {
				if !isNotFoundErr(err) {
					return nil, fmt.Errorf("failed to delete security group %q: %w", g, err)
				}
			}
		}
	}

	if kubernetes.HasFinalizer(cluster, RouterSubnetLinkCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if _, err = detachSubnetFromRouter(netClient, cluster.Spec.Cloud.OpenStack.SubnetID, cluster.Spec.Cloud.OpenStack.RouterID); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to detach subnet from router: %w", err)
			}
		}
	}

	if kubernetes.HasFinalizer(cluster, RouterIPv6SubnetLinkCleanupFinalizer) {
		if _, err = detachSubnetFromRouter(netClient, cluster.Spec.Cloud.OpenStack.IPv6SubnetID, cluster.Spec.Cloud.OpenStack.RouterID); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to detach subnet from router: %w", err)
			}
		}
	}

	if kubernetes.HasFinalizer(cluster, SubnetCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if err := deleteSubnet(netClient, cluster.Spec.Cloud.OpenStack.SubnetID); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete subnet '%s': %w", cluster.Spec.Cloud.OpenStack.SubnetID, err)
			}
		}
	}

	if kubernetes.HasFinalizer(cluster, IPv6SubnetCleanupFinalizer) {
		if err := deleteSubnet(netClient, cluster.Spec.Cloud.OpenStack.IPv6SubnetID); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete subnet '%s': %w", cluster.Spec.Cloud.OpenStack.IPv6SubnetID, err)
			}
		}
	}

	if kubernetes.HasFinalizer(cluster, NetworkCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if err = deleteNetworkByName(netClient, cluster.Spec.Cloud.OpenStack.Network); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete network '%s': %w", cluster.Spec.Cloud.OpenStack.Network, err)
			}
		}
	}

	if kubernetes.HasFinalizer(cluster, RouterCleanupFinalizer) || kubernetes.HasFinalizer(cluster, OldNetworkCleanupFinalizer) {
		if err = deleteRouter(netClient, cluster.Spec.Cloud.OpenStack.RouterID); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete router '%s': %w", cluster.Spec.Cloud.OpenStack.RouterID, err)
			}
		}
	}

	// Relying on the idempotence of the clean-up steps we remove all finalizers in
	// one shot only when the clean-up is completed.
	cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kubernetes.RemoveFinalizer(
			cluster,
			SecurityGroupCleanupFinalizer,
			RouterSubnetLinkCleanupFinalizer,
			RouterIPv6SubnetLinkCleanupFinalizer,
			SubnetCleanupFinalizer,
			IPv6SubnetCleanupFinalizer,
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
func GetFlavors(authURL, region string, credentials *resources.OpenStackCredentials, caBundle *x509.CertPool) ([]osflavors.Flavor, error) {
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
func GetTenants(authURL, region string, credentials *resources.OpenStackCredentials, caBundle *x509.CertPool) ([]osprojects.Project, error) {
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
func GetNetworks(ctx context.Context, authURL, region string, credentials *resources.OpenStackCredentials, caBundle *x509.CertPool) ([]NetworkWithExternalExt, error) {
	authClient, err := getNetClient(ctx, authURL, region, credentials, caBundle)
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
func GetSecurityGroups(ctx context.Context, authURL, region string, credentials *resources.OpenStackCredentials, caBundle *x509.CertPool) ([]ossecuritygroups.SecGroup, error) {
	netClient, err := getNetClient(ctx, authURL, region, credentials, caBundle)
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
func GetAvailabilityZones(authURL, region string, credentials *resources.OpenStackCredentials, caBundle *x509.CertPool) ([]osavailabilityzones.AvailabilityZone, error) {
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

// GetSubnetPools lists all available subnet pools.
func GetSubnetPools(ctx context.Context, authURL, region string, credentials *resources.OpenStackCredentials, ipVersion int, caBundle *x509.CertPool) ([]ossubnetpools.SubnetPool, error) {
	authClient, err := getNetClient(ctx, authURL, region, credentials, caBundle)
	if err != nil {
		return nil, fmt.Errorf("couldn't get auth client: %w", err)
	}

	subnetPools, err := getAllSubnetPools(authClient, ossubnetpools.ListOpts{IPVersion: ipVersion})
	if err != nil {
		return nil, fmt.Errorf("couldn't get subnet pools: %w", err)
	}

	return subnetPools, nil
}

func getAuthClient(authURL string, credentials *resources.OpenStackCredentials, caBundle *x509.CertPool) (*gophercloud.ProviderClient, error) {
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

func getNetClient(ctx context.Context, authURL, region string, credentials *resources.OpenStackCredentials, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
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

	serviceClient.Context = ctx

	return serviceClient, err
}

func getComputeClient(authURL, region string, credentials *resources.OpenStackCredentials, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
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
func GetSubnets(ctx context.Context, authURL, region, networkID string, credentials *resources.OpenStackCredentials, caBundle *x509.CertPool) ([]ossubnets.Subnet, error) {
	serviceClient, err := getNetClient(ctx, authURL, region, credentials, caBundle)
	if err != nil {
		return nil, fmt.Errorf("couldn't get auth client: %w", err)
	}

	subnets, err := getSubnetForNetwork(serviceClient, networkID)
	if err != nil {
		return nil, err
	}

	return subnets, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (os *Provider) ValidateCloudSpecUpdate(_ context.Context, oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	if oldSpec.OpenStack == nil || newSpec.OpenStack == nil {
		return errors.New("'openstack' spec is empty")
	}

	// we validate that a couple of resources are not changed.
	// the exception being the provider itself updating it in case the field
	// was left empty to dynamically generate resources.

	if oldSpec.OpenStack.Network != "" && oldSpec.OpenStack.Network != newSpec.OpenStack.Network {
		return fmt.Errorf("updating OpenStack network is not supported (was %s, updated to %s)", oldSpec.OpenStack.Network, newSpec.OpenStack.Network)
	}

	if oldSpec.OpenStack.SubnetID != "" && oldSpec.OpenStack.SubnetID != newSpec.OpenStack.SubnetID {
		return fmt.Errorf("updating OpenStack subnet ID is not supported (was %s, updated to %s)", oldSpec.OpenStack.SubnetID, newSpec.OpenStack.SubnetID)
	}

	if oldSpec.OpenStack.RouterID != "" && oldSpec.OpenStack.RouterID != newSpec.OpenStack.RouterID {
		return fmt.Errorf("updating OpenStack router ID is not supported (was %s, updated to %s)", oldSpec.OpenStack.RouterID, newSpec.OpenStack.RouterID)
	}

	if oldSpec.OpenStack.SecurityGroups != "" && oldSpec.OpenStack.SecurityGroups != newSpec.OpenStack.SecurityGroups {
		return fmt.Errorf("updating OpenStack security groups is not supported (was %s, updated to %s)", oldSpec.OpenStack.SecurityGroups, newSpec.OpenStack.SecurityGroups)
	}

	return nil
}

func getNetClientForCluster(ctx context.Context, cluster kubermaticv1.CloudSpec, dc *kubermaticv1.DatacenterSpecOpenStack, secretKeySelector provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
	creds, err := GetCredentialsForCluster(cluster, secretKeySelector)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	netClient, err := getNetClient(ctx, dc.AuthURL, dc.Region, creds, caBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to create a authenticated openstack client: %w", err)
	}
	return netClient, nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
// The user can choose three ways for authentication. The first is a token. Second through Application Credentials.
// The last one uses a username and password. Those methods work exclusively.
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (*resources.OpenStackCredentials, error) {
	username := cloud.OpenStack.Username
	password := cloud.OpenStack.Password
	project := cloud.OpenStack.Project
	projectID := cloud.OpenStack.ProjectID
	domain := cloud.OpenStack.Domain
	applicationCredentialID := cloud.OpenStack.ApplicationCredentialID
	applicationCredentialSecret := cloud.OpenStack.ApplicationCredentialSecret
	useToken := cloud.OpenStack.UseToken
	token := cloud.OpenStack.Token
	var err error

	if applicationCredentialID != "" && applicationCredentialSecret != "" {
		return &resources.OpenStackCredentials{
			ApplicationCredentialSecret: applicationCredentialSecret,
			ApplicationCredentialID:     applicationCredentialID,
		}, nil
	}

	if applicationCredentialID == "" && cloud.OpenStack.CredentialsReference != nil {
		applicationCredentialID, _ = secretKeySelector(cloud.OpenStack.CredentialsReference, resources.OpenstackApplicationCredentialID)
		if applicationCredentialID != "" {
			applicationCredentialSecret, err = secretKeySelector(cloud.OpenStack.CredentialsReference, resources.OpenstackApplicationCredentialSecret)
			if err != nil {
				return &resources.OpenStackCredentials{}, err
			}

			return &resources.OpenStackCredentials{
				ApplicationCredentialSecret: applicationCredentialSecret,
				ApplicationCredentialID:     applicationCredentialID,
			}, nil
		}
	}

	if domain == "" {
		if cloud.OpenStack.CredentialsReference == nil {
			return &resources.OpenStackCredentials{}, errors.New("no credentials provided")
		}
		domain, err = secretKeySelector(cloud.OpenStack.CredentialsReference, resources.OpenstackDomain)
		if err != nil {
			return &resources.OpenStackCredentials{}, err
		}
	}

	if useToken && token != "" {
		return &resources.OpenStackCredentials{
			Token:  token,
			Domain: domain,
		}, nil
	}

	if !useToken && cloud.OpenStack.CredentialsReference != nil {
		token, _ := secretKeySelector(cloud.OpenStack.CredentialsReference, resources.OpenstackToken)
		if token != "" {
			return &resources.OpenStackCredentials{
				Token:  token,
				Domain: domain,
			}, nil
		}
	}

	if username == "" {
		if cloud.OpenStack.CredentialsReference == nil {
			return &resources.OpenStackCredentials{}, errors.New("no credentials provided")
		}
		username, err = secretKeySelector(cloud.OpenStack.CredentialsReference, resources.OpenstackUsername)
		if err != nil {
			return &resources.OpenStackCredentials{}, err
		}
	}

	if password == "" {
		if cloud.OpenStack.CredentialsReference == nil {
			return &resources.OpenStackCredentials{}, errors.New("no credentials provided")
		}
		password, err = secretKeySelector(cloud.OpenStack.CredentialsReference, resources.OpenstackPassword)
		if err != nil {
			return &resources.OpenStackCredentials{}, err
		}
	}

	if project == "" && cloud.OpenStack.CredentialsReference != nil && cloud.OpenStack.CredentialsReference.Name != "" {
		if project, err = firstKey(secretKeySelector, cloud.OpenStack.CredentialsReference, resources.OpenstackProject, resources.OpenstackTenant); err != nil {
			return &resources.OpenStackCredentials{}, err
		}
	}

	if projectID == "" && cloud.OpenStack.CredentialsReference != nil && cloud.OpenStack.CredentialsReference.Name != "" {
		if projectID, err = firstKey(secretKeySelector, cloud.OpenStack.CredentialsReference, resources.OpenstackProjectID, resources.OpenstackTenantID); err != nil {
			return &resources.OpenStackCredentials{}, err
		}
	}

	return &resources.OpenStackCredentials{
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
func firstKey(secretKeySelector provider.SecretKeySelectorValueFunc, configVar *kubermaticv1.GlobalSecretKeySelector, firstKey string, fallbackKey string) (string, error) {
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

func ValidateCredentials(authURL, region string, credentials *resources.OpenStackCredentials, caBundle *x509.CertPool) error {
	computeClient, err := getComputeClient(authURL, region, credentials, caBundle)
	if err != nil {
		return err
	}
	_, err = getAvailabilityZones(computeClient)

	return err
}

func DescribeFlavor(credentials *resources.OpenStackCredentials, authURL, region string, caBundle *x509.CertPool, flavorName string) (*provider.NodeCapacity, error) {
	flavors, err := GetFlavors(authURL, region, credentials, caBundle)
	if err != nil {
		return nil, err
	}

	for _, flavor := range flavors {
		if strings.EqualFold(flavor.Name, flavorName) {
			capacity := provider.NewNodeCapacity()
			capacity.WithCPUCount(flavor.VCPUs)

			if err := capacity.WithMemory(flavor.RAM, "M"); err != nil {
				return nil, fmt.Errorf("failed to parse memory size: %w", err)
			}

			if err := capacity.WithStorage(flavor.Disk, "G"); err != nil {
				return nil, fmt.Errorf("failed to parse disk size: %w", err)
			}

			return capacity, nil
		}
	}

	return nil, fmt.Errorf("cannot find flavor %q", flavorName)
}
