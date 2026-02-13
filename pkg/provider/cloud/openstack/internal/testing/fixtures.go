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

package testing

import (
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

// IDs used when a new resource is created.
// Note that using always the same ID is not correct, but is sufficient to
// cover the current usage. For more advanced usages a pseudo-random UID
// generator should be used instead.
const (
	NetworkID       = "8bb661f5-76b9-45f1-9ef9-eeffcd025fe4"
	SubnetID        = "bec43a98-2d0a-4b1d-9df0-0f21e8f89d8a"
	SubnetPoolID    = "bed43a98-2d0a-4b1d-9df0-0f21e8f89d8a"
	RouterID        = "b8a25073-b35f-4b64-b205-6d21c8221d98"
	PortID          = "6f5c4730-aa3f-4677-bca2-3b1ead8545d9"
	InterfaceInfoID = "6f5c4730-aa3f-4677-bca2-3b1ead8545d9"
	SecGroupID      = "babf744f-0098-4f61-8e15-f91c30b3ed05"
	SecGroupRuleID  = "da8d0577-eeb8-4351-b543-4bb0687f81d3"
)

// Relative paths to resource endpoints.
const (
	SecurityGroupsEndpoint = "/security-groups"
	NetworksEndpoint       = "/networks"
	SubnetsEndpoint        = "/subnets"
	RoutersEndpoint        = "/routers"
	SubnetPoolsEndpoint    = "/subnetpools"
)

func AddRouterInterfaceEndpoint(routerID string) string {
	return "/routers/" + routerID + "/add_router_interface"
}

var ExternalNetwork = Network{
	Network:            networks.Network{Name: "external-network", ID: "d32019d3-bc6e-4319-9c1d-6722fc136a22"},
	NetworkExternalExt: external.NetworkExternalExt{External: true},
}

var ExternalNetworkFoo = Network{
	Network:            networks.Network{Name: "foo", ID: "59618382-5aba-49c5-b415-aabb79708098"},
	NetworkExternalExt: external.NetworkExternalExt{External: true},
}

var SecondExternalNetwork = Network{
	Network:            networks.Network{Name: "second-external-network", ID: "f5e7a7b6-1234-5678-90ab-cdef12345678"},
	NetworkExternalExt: external.NetworkExternalExt{External: true},
}

var InternalNetwork = Network{
	Network: networks.Network{Name: "kubernetes-cluster-xyz", ID: NetworkID},
}

var SubnetTest = Subnet{
	Subnet: subnets.Subnet{ID: SubnetID, Name: "kubernetes-cluster-xyz"},
}
