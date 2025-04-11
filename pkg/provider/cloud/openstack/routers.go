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
	osrouters "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
)

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
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Extract()
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
