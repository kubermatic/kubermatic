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
	osflavors "github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	ossubnets "github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/pagination"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/machine-controller/sdk/providerconfig"
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

	// FloatingIPPoolIDAnnotation stores the ID of the floating IP pool (external network).
	FloatingIPPoolIDAnnotation = "kubermatic.k8c.io/openstack-floating-ip-pool-id"
)

type getClientFunc func(ctx context.Context, cluster kubermaticv1.CloudSpec, dc *kubermaticv1.DatacenterSpecOpenstack, secretKeySelector provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error)

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

var _ provider.ReconcilingCloudProvider = &Provider{}

// DefaultCloudSpec adds defaults to the cloud spec.
func (os *Provider) DefaultCloudSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec) error {
	if spec.Cloud.Openstack == nil {
		return errors.New("no Openstack cloud spec found")
	}

	if spec.Cloud.Openstack.NodePortsAllowedIPRanges == nil {
		switch spec.ClusterNetwork.IPFamily {
		case kubermaticv1.IPFamilyIPv4:
			spec.Cloud.Openstack.NodePortsAllowedIPRanges = &kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{resources.IPv4MatchAnyCIDR},
			}
		case kubermaticv1.IPFamilyDualStack:
			spec.Cloud.Openstack.NodePortsAllowedIPRanges = &kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{resources.IPv4MatchAnyCIDR, resources.IPv6MatchAnyCIDR},
			}
		}
	}

	spec.Cloud.Openstack.CinderTopologyEnabled = os.dc.CSICinderTopologyEnabled
	return nil
}

func (os *Provider) ClusterNeedsReconciling(cluster *kubermaticv1.Cluster) bool {
	return false
}

// ValidateCloudSpec validates the given CloudSpec.
func (os *Provider) ValidateCloudSpec(ctx context.Context, spec kubermaticv1.CloudSpec) error {
	netClient, err := os.getClientFunc(ctx, spec, os.dc, os.secretKeySelector, os.caBundle)
	if err != nil {
		return err
	}

	if spec.Openstack.SecurityGroups != "" {
		if err := validateSecurityGroupExists(netClient, spec.Openstack.SecurityGroups); err != nil {
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

	if spec.Openstack.IPv6SubnetPool != "" {
		subnetPool, err := getSubnetPoolByName(netClient, spec.Openstack.IPv6SubnetPool)
		if err != nil {
			return fmt.Errorf("failed to get subnet pool %q: %w", spec.Openstack.IPv6SubnetPool, err)
		}
		if subnetPool.IPversion != 6 {
			return fmt.Errorf("provided IPv6 subnet pool %q has incorrect IP version: %d", spec.Openstack.IPv6SubnetPool, subnetPool.IPversion)
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

// InitializeCloudProvider initializes a cluster, in particular
// creates security group and network configuration.
func (os *Provider) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return os.reconcileCluster(ctx, cluster, update, false)
}

// ReconcileCluster reconcile the cluster resources
// reconcile network, security group, subnets, routers and attach routers to subnet.
func (os *Provider) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return os.reconcileCluster(ctx, cluster, update, true)
}

func (os *Provider) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, force bool) (*kubermaticv1.Cluster, error) {
	netClient, err := os.getClientFunc(ctx, cluster.Spec.Cloud, os.dc, os.secretKeySelector, os.caBundle)
	if err != nil {
		return nil, err
	}

	// Reconciling the external Network (the floating IP pool used for machines and LBs)
	// We don't need the usual if conditional here because the reconcile function doesn't
	// create anything.
	cluster, err = reconcileExtNetwork(ctx, netClient, cluster, update)
	if err != nil {
		return nil, err
	}

	// Reconciling the Network
	if force || cluster.Spec.Cloud.Openstack.Network == "" {
		cluster, err = reconcileNetwork(ctx, netClient, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	// Reconcile the security group(s)
	if force || cluster.Spec.Cloud.Openstack.SecurityGroups == "" {
		cluster, err = os.reconcileSecurityGroups(ctx, netClient, cluster, update)
		if err != nil {
			return nil, err
		}
	}

	// Reconciling the subnets. All machines will live in one dedicated subnet.
	if force || cluster.Spec.Cloud.Openstack.SubnetID == "" || cluster.Spec.Cloud.Openstack.IPv6SubnetID == "" {
		network, err := getNetworkByName(netClient, cluster.Spec.Cloud.Openstack.Network, false)
		if err != nil {
			return nil, fmt.Errorf("failed to get network '%s': %w", cluster.Spec.Cloud.Openstack.Network, err)
		}
		cluster, err = reconcileIPv4Subnet(ctx, netClient, cluster, update, network.ID, os.dc.DNSServers)
		if err != nil {
			return nil, err
		}
		cluster, err = reconcileIPv6Subnet(ctx, netClient, cluster, update, network.ID, os.dc.DNSServers)
		if err != nil {
			return nil, err
		}
	}
	if force || cluster.Spec.Cloud.Openstack.RouterID == "" {
		cluster, err = reconcileRouter(ctx, netClient, cluster, update)
		if err != nil {
			return nil, err
		}
	}
	return cluster, nil
}

func reconcileNetwork(ctx context.Context, netClient *gophercloud.ServiceClient, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	networkName := cluster.Spec.Cloud.Openstack.Network

	// If the network name is set, check if it exists
	if networkName != "" {
		existingNetwork, err := getNetworkByName(netClient, networkName, false)
		if err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to check for existing network: %w", err)
			}
		}
		if existingNetwork != nil {
			return cluster, nil
		}
	} else {
		// If NetworkName not specified, Create network with name kubernetes-clusterid.
		networkName = resourceNamePrefix + cluster.Name
	}

	cluster, err := update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kubernetes.AddFinalizer(cluster, NetworkCleanupFinalizer)
		cluster.Spec.Cloud.Openstack.Network = networkName
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update network cluster: %w", err)
	}

	_, err = createUserClusterNetwork(netClient, networkName)

	if err != nil {
		return nil, fmt.Errorf("failed to create the network: %w", err)
	}

	return cluster, nil
}

func reconcileExtNetwork(ctx context.Context, netClient *gophercloud.ServiceClient, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var (
		err        error
		extNetwork *NetworkWithExternalExt
	)

	if cluster.Spec.Cloud.Openstack.FloatingIPPool == "" {
		// Fetch the default external network if no floating IP pool was provided.
		extNetwork, err = getDefaultExternalNetwork(netClient)
		if err != nil {
			return nil, fmt.Errorf("failed to get external network for floating IP pool: %w", err)
		}
	} else {
		// Fetch the configured external network by name if a floating IP pool is provided.
		extNetwork, err = getNetworkByName(netClient, cluster.Spec.Cloud.Openstack.FloatingIPPool, true)
		if err != nil {
			return nil, fmt.Errorf("failed to get external network for floating IP pool by name: %w", err)
		}
	}

	// We're just searching for the floating ip pool here & don't create anything. Thus no need to create a finalizer.
	cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		// This should be a noop if the floating IP pool was already correctly provided.
		cluster.Spec.Cloud.Openstack.FloatingIPPool = extNetwork.Name

		if cluster.Annotations == nil {
			cluster.Annotations = make(map[string]string)
		}
		cluster.Annotations[FloatingIPPoolIDAnnotation] = extNetwork.ID
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update cluster floating IP pool: %w", err)
	}

	return cluster, err
}

func (os *Provider) reconcileSecurityGroups(ctx context.Context, netClient *gophercloud.ServiceClient, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	securityGroup := cluster.Spec.Cloud.Openstack.SecurityGroups

	// automatically create and fill-in the default security group if none was specified
	if securityGroup == "" {
		securityGroup = resourceNamePrefix + cluster.Name
	}

	ipv4Network := cluster.IsIPv4Only() || cluster.IsDualStack()
	ipv6Network := cluster.IsIPv6Only() || cluster.IsDualStack()

	ipRanges := resources.GetNodePortsAllowedIPRanges(cluster, cluster.Spec.Cloud.Openstack.NodePortsAllowedIPRanges, cluster.Spec.Cloud.Openstack.NodePortsAllowedIPRange, os.dc.NodePortsAllowedIPRanges)

	lowPort, highPort := resources.NewTemplateDataBuilder().
		WithNodePortRange(cluster.Spec.ComponentsOverride.Apiserver.NodePortRange).
		WithCluster(cluster).
		Build().
		NodePorts()

	// for each security group, ensure that it exists
	err := validateSecurityGroupExists(netClient, securityGroup)

	if err != nil {
		if isNotFoundErr(err) {
			// group does not yet exist, so we create it
			req := securityGroupSpec{
				name:           securityGroup,
				ipv4Rules:      ipv4Network,
				ipv6Rules:      ipv6Network,
				lowPort:        lowPort,
				highPort:       highPort,
				nodePortsCIDRs: ipRanges,
			}

			_, err = ensureSecurityGroup(netClient, req)
			if err != nil {
				return nil, fmt.Errorf("failed to create security group: %w", err)
			}

			cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
				kubernetes.AddFinalizer(cluster, SecurityGroupCleanupFinalizer)
			})

			if err != nil {
				return nil, fmt.Errorf("failed to add security group finalizer: %w", err)
			}
		} else {
			return cluster, fmt.Errorf("failed to check security group: %w", err)
		}
	}

	cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		cluster.Spec.Cloud.Openstack.SecurityGroups = securityGroup
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update security group in cluster: %w", err)
	}

	return cluster, nil
}

func reconcileIPv6Subnet(ctx context.Context, netClient *gophercloud.ServiceClient, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, network string, dnservers []string) (*kubermaticv1.Cluster, error) {
	ipv6Network := cluster.IsIPv6Only() || cluster.IsDualStack()
	if ipv6Network {
		if cluster.Spec.Cloud.Openstack.IPv6SubnetID != "" {
			// The Subnet ID is specified, try to find the subnet by ID
			_, err := getSubnetByID(netClient, cluster.Spec.Cloud.Openstack.IPv6SubnetID)
			if err != nil {
				if !isNotFoundErr(err) {
					return nil, fmt.Errorf("failed to get subnet by ID: %w", err)
				}
			} else {
				return cluster, nil
			}
		}

		// Subnet not found by ID, try finding by name
		subnet, err := getSubnetByName(netClient, resourceNamePrefix+cluster.Name+"-ipv6")
		if err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to get subnet by name: %w", err)
			}
		} else {
			// Subnet found by name, update the cluster spec with the found subnet ID and return
			// Update the cluster spec with the new subnet ID
			cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
				cluster.Spec.Cloud.Openstack.IPv6SubnetID = subnet.ID
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update IPv6 subnet to the cluster: %w", err)
			}
			return cluster, nil
		}
		// At this point, either the SubnetID was empty, or the specified subnet was not found by ID or name
		// Proceed to create a new subnet
		subnet, err = createIPv6Subnet(netClient, cluster.Name, network, cluster.Spec.Cloud.Openstack.IPv6SubnetPool, dnservers)
		if err != nil {
			return nil, fmt.Errorf("failed to create the IPv6 subnet: %w", err)
		}
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.AddFinalizer(cluster, IPv6SubnetCleanupFinalizer, RouterIPv6SubnetLinkCleanupFinalizer)
			cluster.Spec.Cloud.Openstack.IPv6SubnetID = subnet.ID
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update IPv6 subnet ID: %w", err)
		}
	}
	return cluster, nil
}

func reconcileIPv4Subnet(ctx context.Context, netClient *gophercloud.ServiceClient, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, network string, dnservers []string) (*kubermaticv1.Cluster, error) {
	ipv4Network := cluster.IsIPv4Only() || cluster.IsDualStack()

	if ipv4Network {
		if cluster.Spec.Cloud.Openstack.SubnetID != "" {
			// The Subnet ID is specified, try to find the subnet by ID
			_, err := getSubnetByID(netClient, cluster.Spec.Cloud.Openstack.SubnetID)
			if err != nil {
				if !isNotFoundErr(err) {
					return nil, fmt.Errorf("failed to get subnet by ID: %w", err)
				}
			} else {
				// Subnet found by ID, no action needed, return the cluster
				return cluster, nil
			}
		}

		// Subnet not found by ID, try finding by name
		subnet, err := getSubnetByName(netClient, resourceNamePrefix+cluster.Name)
		if err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to get subnet by name: %w", err)
			}
		} else {
			// Subnet found by name, update the cluster spec with the found subnet ID and return
			// Update the cluster spec with the new subnet ID
			cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
				cluster.Spec.Cloud.Openstack.SubnetID = subnet.ID
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update IPv4 subnet to the cluster: %w", err)
			}
			return cluster, nil
		}
		// At this point, either the SubnetID was empty, or the specified subnet was not found by ID or name
		// Proceed to create a new subnet
		subnet, err = createSubnet(netClient, cluster.Name, network, dnservers)
		if err != nil {
			return nil, fmt.Errorf("failed to create the IPv4 subnet: %w", err)
		}

		// Update the cluster spec with the new subnet ID
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.AddFinalizer(cluster, SubnetCleanupFinalizer, RouterSubnetLinkCleanupFinalizer)
			cluster.Spec.Cloud.Openstack.SubnetID = subnet.ID
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update IPv4 subnet to the cluster: %w", err)
		}
	}
	return cluster, nil
}

func reconcileRouter(ctx context.Context, netClient *gophercloud.ServiceClient, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if cluster.Spec.Cloud.Openstack.RouterID != "" {
		// Check if the router exists
		router, err := getRouterByID(netClient, cluster.Spec.Cloud.Openstack.RouterID)
		if err != nil {
			if !isNotFoundErr(err) {
				return nil, err
			}
		}
		if router != nil {
			// Router found, attach subnets if not already attached
			err = attachSubnetsIfNeeded(ctx, netClient, cluster, update)
			return cluster, err
		}
	}
	// If RouterID is empty, try to find the router by name
	router, err := getRouterByName(netClient, resourceNamePrefix+cluster.Name)
	if err != nil {
		if !isNotFoundErr(err) {
			return nil, fmt.Errorf("failed to get router by name: %w", err)
		}
	} else if router != nil {
		// Found an existing router by name, update the cluster spec with this RouterID
		// Update the cluster spec with the new router ID
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.RouterID = router.ID
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update RouterID in the cluster spec: %w", err)
		}
		err = attachSubnetsIfNeeded(ctx, netClient, cluster, update)
		return cluster, err
	}
	var routerID string
	// if SubnetID is provided but RouterID not, try to retrieve RouterID
	if cluster.Spec.Cloud.Openstack.SubnetID != "" {
		var err error
		routerID, err = getRouterIDForSubnet(netClient, cluster.Spec.Cloud.Openstack.SubnetID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify that the subnet '%s' has a router attached: %w", cluster.Spec.Cloud.Openstack.SubnetID, err)
		}
	}
	if cluster.Spec.Cloud.Openstack.IPv6SubnetID != "" && routerID == "" {
		var err error
		routerID, err = getRouterIDForSubnet(netClient, cluster.Spec.Cloud.Openstack.IPv6SubnetID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify that the subnet '%s' has a router attached: %w", cluster.Spec.Cloud.Openstack.IPv6SubnetID, err)
		}
	}
	if routerID != "" {
		// Update the cluster spec with the new router ID
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.Openstack.RouterID = routerID
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update RouterID in the cluster spec: %w", err)
		}
		return cluster, nil
	}
	// Router not found by name, create a new router
	router, err = createRouter(netClient, cluster.Name, cluster.Spec.Cloud.Openstack.FloatingIPPool)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new router: %w", err)
	}

	// Update the cluster spec with the new router ID
	cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kubernetes.AddFinalizer(cluster, RouterCleanupFinalizer)
		cluster.Spec.Cloud.Openstack.RouterID = router.ID
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update RouterID in the cluster spec: %w", err)
	}

	// Attach the new router to subnets
	err = attachSubnetsIfNeeded(ctx, netClient, cluster, update)

	return cluster, err
}

func attachSubnetsIfNeeded(ctx context.Context, netClient *gophercloud.ServiceClient, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) error {
	// Check if the router is already attached to the IPv4 subnet and attach to missing subnet if necessary
	if cluster.Spec.Cloud.Openstack.SubnetID != "" {
		err := linkSubnetToRouter(ctx, netClient, cluster, cluster.Spec.Cloud.Openstack.SubnetID, cluster.Spec.Cloud.Openstack.RouterID, RouterSubnetLinkCleanupFinalizer, update)
		if err != nil {
			return err
		}
	}

	// Check if the router is already attached to the IPv6 subnetand attach to missing subnet if necessary
	if cluster.Spec.Cloud.Openstack.IPv6SubnetID != "" {
		err := linkSubnetToRouter(ctx, netClient, cluster, cluster.Spec.Cloud.Openstack.IPv6SubnetID, cluster.Spec.Cloud.Openstack.RouterID, RouterIPv6SubnetLinkCleanupFinalizer, update)
		if err != nil {
			return err
		}
	}

	return nil
}

func linkSubnetToRouter(ctx context.Context, netClient *gophercloud.ServiceClient, cluster *kubermaticv1.Cluster, subnetID string, routerID string, finalizer string, update provider.ClusterUpdater) error {
	var ipAttached bool
	router, err := getRouterIDForSubnet(netClient, subnetID)
	ipAttached = err == nil && router != ""
	if !ipAttached {
		_, err := attachSubnetToRouter(netClient, subnetID, routerID)
		if err != nil {
			return err
		}
		// Add the Link finalizer
		_, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kubernetes.AddFinalizer(cluster, finalizer)
		})
		if err != nil {
			return fmt.Errorf("failed to add %s in the cluster spec: %w", finalizer, err)
		}
	}

	return nil
}

// CleanUpCloudProvider does the clean-up in particular:
// removes security group and network configuration.
func (os *Provider) CleanUpCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	netClient, err := os.getClientFunc(ctx, cluster.Spec.Cloud, os.dc, os.secretKeySelector, os.caBundle)
	if err != nil {
		return nil, err
	}

	if kubernetes.HasFinalizer(cluster, SecurityGroupCleanupFinalizer) {
		sg := cluster.Spec.Cloud.Openstack.SecurityGroups
		if err := deleteSecurityGroup(netClient, sg); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete security group %q: %w", sg, err)
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

	if kubernetes.HasFinalizer(cluster, RouterIPv6SubnetLinkCleanupFinalizer) {
		if _, err = detachSubnetFromRouter(netClient, cluster.Spec.Cloud.Openstack.IPv6SubnetID, cluster.Spec.Cloud.Openstack.RouterID); err != nil {
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

	if kubernetes.HasFinalizer(cluster, IPv6SubnetCleanupFinalizer) {
		if err := deleteSubnet(netClient, cluster.Spec.Cloud.Openstack.IPv6SubnetID); err != nil {
			if !isNotFoundErr(err) {
				return nil, fmt.Errorf("failed to delete subnet '%s': %w", cluster.Spec.Cloud.Openstack.IPv6SubnetID, err)
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

func getNetClient(ctx context.Context, authURL, region string, credentials *resources.OpenstackCredentials, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
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

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (os *Provider) ValidateCloudSpecUpdate(_ context.Context, oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	if oldSpec.Openstack == nil || newSpec.Openstack == nil {
		return errors.New("'openstack' spec is empty")
	}

	// we validate that a couple of resources are not changed.
	// the exception being the provider itself updating it in case the field
	// was left empty to dynamically generate resources.

	if oldSpec.Openstack.Network != "" && oldSpec.Openstack.Network != newSpec.Openstack.Network {
		return fmt.Errorf("updating OpenStack network is not supported (was %s, updated to %s)", oldSpec.Openstack.Network, newSpec.Openstack.Network)
	}

	if oldSpec.Openstack.SubnetID != "" && oldSpec.Openstack.SubnetID != newSpec.Openstack.SubnetID {
		return fmt.Errorf("updating OpenStack subnet ID is not supported (was %s, updated to %s)", oldSpec.Openstack.SubnetID, newSpec.Openstack.SubnetID)
	}

	if oldSpec.Openstack.SecurityGroups != "" && oldSpec.Openstack.SecurityGroups != newSpec.Openstack.SecurityGroups {
		if isMultipleSGs(oldSpec.Openstack.SecurityGroups) && !isMultipleSGs(newSpec.Openstack.SecurityGroups) {
			return nil
		}

		return fmt.Errorf("updating OpenStack security group is not supported; only migration from multiple (comma-separated) security groups to a single security group is allowed (was %s, updated to %s)", oldSpec.Openstack.SecurityGroups, newSpec.Openstack.SecurityGroups)
	}

	return nil
}

func getNetClientForCluster(ctx context.Context, cluster kubermaticv1.CloudSpec, dc *kubermaticv1.DatacenterSpecOpenstack, secretKeySelector provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (*gophercloud.ServiceClient, error) {
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

	if applicationCredentialID != "" && applicationCredentialSecret != "" {
		return &resources.OpenstackCredentials{
			ApplicationCredentialSecret: applicationCredentialSecret,
			ApplicationCredentialID:     applicationCredentialID,
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
			}, nil
		}
	}

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

func DescribeFlavor(credentials *resources.OpenstackCredentials, authURL, region string, caBundle *x509.CertPool, flavorName string) (*provider.NodeCapacity, error) {
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
