/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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
	"errors"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud"
	tags "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	osrouters "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
)

// createRouter creates a Neutron router and ensures it is tagged correctly.
// The router is considered invalid without proper tags:
//   - OpenStack API does not support creating a router with tags directly;
//     we must first create the router and then tag the resource.
//   - If tagging fails, the router is deleted to prevent orphaned resources.
//   - If deletion after tagging failure also fails, it's a critical error requiring manual intervention.
func createRouter(netClient *gophercloud.ServiceClient, clusterName, extNetworkName string) (*osrouters.Router, error) {
	extNetwork, err := getNetworkByName(netClient, extNetworkName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get external network %q: %w", extNetworkName, err)
	}

	iTrue := true
	gwi := osrouters.GatewayInfo{
		NetworkID: extNetwork.ID,
	}

	res := osrouters.Create(netClient, osrouters.CreateOpts{
		Name:         resourceNamePrefix + clusterName,
		AdminStateUp: &iTrue,
		GatewayInfo:  &gwi,
	})

	router, err := res.Extract()
	if err != nil {
		return nil, err
	}

	// CRITICAL SECTION: Tagging is mandatory for resource management.
	// - Tags enable lifecycle tracking (e.g., cleanup, cost allocation)
	// - Retry tagging up to 3 times with exponential backoff for transient failures
	err = addTags(netClient, clusterName, router.ID)
	if err != nil {
		// FALLBACK: Delete the router if tagging fails
		// - Prevents orphaned routers without required metadata
		// - Retry deletion 3 times with exponential backoff (transient failures possible)
		deleteErr := retryOnError(3, func() error { return deleteRouter(netClient, router.ID) })
		if deleteErr != nil {
			return nil, fmt.Errorf(
				"CRITICAL FAILURE: Router %s created but tagging failed, and deletion also failed: %w (original error: %w)",
				router.ID, deleteErr, err,
			)
		}
		return nil, fmt.Errorf("failed to tag router (rolled back): %w", err)
	}

	return router, nil
}

func getRouterByName(netClient *gophercloud.ServiceClient, name string) (*osrouters.Router, error) {
	routers, err := listRouters(netClient, osrouters.ListOpts{Name: name})
	if err != nil {
		return nil, err
	}
	switch len(routers) {
	case 1:
		return &routers[0], nil
	case 0:
		return nil, fmt.Errorf("router with name '%s' not found", name)
	default:
		return nil, fmt.Errorf("found %d routers for name '%s', expected exactly one", len(routers), name)
	}
}

func getRouterByID(netClient *gophercloud.ServiceClient, id string) (*osrouters.Router, error) {
	routers, err := listRouters(netClient, osrouters.ListOpts{ID: id})
	if err != nil {
		return nil, err
	}
	switch len(routers) {
	case 1:
		return &routers[0], nil
	case 0:
		return nil, fmt.Errorf("router with ID '%s' not found", id)
	default:
		return nil, fmt.Errorf("found %d routers for ID '%s', expected exactly one", len(routers), id)
	}
}

func listRouters(netClient *gophercloud.ServiceClient, listOpts osrouters.ListOpts) ([]osrouters.Router, error) {
	allPages, err := osrouters.List(netClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	allRouters, err := osrouters.ExtractRouters(allPages)
	if err != nil {
		return nil, err
	}
	return allRouters, nil
}

func getRouterIDForSubnet(netClient *gophercloud.ServiceClient, subnetID string) (string, error) {
	ports, err := getAllNetworkPorts(netClient, subnetID)
	if err != nil {
		return "", fmt.Errorf("failed to list ports for subnet: %w", err)
	}

	for _, port := range ports {
		if port.DeviceOwner == "network:router_interface" || port.DeviceOwner == "network:router_interface_distributed" || port.DeviceOwner == "network:ha_router_replicated_interface" {
			// Check IP for the interface & check if the IP belongs to the subnet
			return port.DeviceID, nil
		}
	}

	return "", nil
}

func deleteRouter(netClient *gophercloud.ServiceClient, routerID string) error {
	res := osrouters.Delete(netClient, routerID)
	if res.Err != nil {
		return res.Err
	}
	return res.ExtractErr()
}

func ignoreRouterAlreadyHasPortInSubnetError(err error, subnetID string) error {
	matchString := fmt.Sprintf("Router already has a port on subnet %s", subnetID)

	var gopherCloud400Err gophercloud.ErrDefault400
	if !errors.As(err, &gopherCloud400Err) || !strings.Contains(string(gopherCloud400Err.Body), matchString) {
		return err
	}

	return nil
}

func addTags(netClient *gophercloud.ServiceClient, clusterName, routerID string) error {
	return retryOnError(3, func() error {
		tagOpts := tags.ReplaceAllOpts{
			Tags: []string{
				TagManagedByKubermatic,
				TagPrefixClusterID + clusterName,
			},
		}
		return tags.ReplaceAll(netClient, ResourceTypeRouter, routerID, tagOpts).Err
	})
}

// isManagedRouter checks if a router is managed by Kubermatic KKP.
func isManagedRouter(netClient *gophercloud.ServiceClient, routerID string) bool {
	return isManagedResource(netClient, ResourceTypeRouter, routerID)
}
func ownTheRouter(netClient *gophercloud.ServiceClient, routerID string, clusterName string) error {
	return addOwnershipToResource(netClient, ResourceTypeRouter, routerID, clusterName)
}

func removerRouterOwnership(netClient *gophercloud.ServiceClient, routerID string, clusterName string) error {
	return removeOwnershipFromResource(netClient, ResourceTypeRouter, routerID, clusterName)
}
func getRouterOwners(netClient *gophercloud.ServiceClient, routerID string) ([]string, error) {
	return getResourceOwners(netClient, ResourceTypeRouter, routerID)
}
