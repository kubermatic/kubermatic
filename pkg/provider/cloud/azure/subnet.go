/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package azure

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/ptr"
)

const (
	defaultSubnetCIDRIPv4 = "10.0.0.0/16"
	defaultSubnetCIDRIPv6 = "fd00::/64"
)

func subnetName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileSubnet(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if cluster.Spec.Cloud.Azure.SubnetName == "" {
		cluster.Spec.Cloud.Azure.SubnetName = subnetName(cluster)
	}

	resourceGroup := getResourceGroup(cluster.Spec.Cloud)

	vnet, err := clients.Networks.Get(ctx, resourceGroup, cluster.Spec.Cloud.Azure.VNetName, nil)
	if err != nil && !isNotFound(err) {
		return nil, err
	}

	routeTable, err := clients.RouteTables.Get(ctx, cluster.Spec.Cloud.Azure.ResourceGroup, cluster.Spec.Cloud.Azure.RouteTableName, nil)
	if err != nil && !isNotFound(err) {
		return nil, err
	}

	subnet, err := clients.Subnets.Get(ctx, resourceGroup, *vnet.Name, cluster.Spec.Cloud.Azure.SubnetName, nil)
	if err != nil && !isNotFound(err) {
		return nil, err
	}

	// since subnets are sub-resources of VNETs and don't have tags themselves
	// we can only guess KKP ownership based on the VNET ownership tag. If the
	// VNET isn't owned by KKP, we should not try to reconcile subnets and
	// return early.
	if err == nil && !hasOwnershipTag(vnet.Tags, cluster) {
		return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.SubnetName = cluster.Spec.Cloud.Azure.SubnetName
		})
	}

	var cidrs []string
	if cluster.IsIPv4Only() || cluster.IsDualStack() {
		cidrs = append(cidrs, defaultSubnetCIDRIPv4)
	}
	if cluster.IsIPv6Only() || cluster.IsDualStack() {
		cidrs = append(cidrs, defaultSubnetCIDRIPv6)
	}

	target := targetSubnet(cluster.Spec.Cloud, routeTable.ID, cidrs)

	// check for attributes of the existing subnet and skip ensuring if all values are already
	// as expected. Since there are a lot of pointers in the network.Subnet struct, we need to
	// do a lot of "!= nil" checks so this does not panic.
	//
	// Attributes we check:
	// - Subnet CIDR
	if subnet.Properties == nil ||
		!reflect.DeepEqual(subnet.Properties.AddressPrefix, target.Properties.AddressPrefix) ||
		!reflect.DeepEqual(subnet.Properties.AddressPrefixes, target.Properties.AddressPrefixes) ||
		!reflect.DeepEqual(subnet.Properties.RouteTable, target.Properties.RouteTable) {
		if err := ensureSubnet(ctx, clients, cluster.Spec.Cloud, target); err != nil {
			return nil, err
		}
	}

	return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.SubnetName = cluster.Spec.Cloud.Azure.SubnetName
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerSubnet)
	})
}

func targetSubnet(cloud kubermaticv1.CloudSpec, routeTableID *string, cidrs []string) *armnetwork.Subnet {
	cidrPointers := []*string{}
	for _, cidr := range cidrs {
		cidrPointers = append(cidrPointers, ptr.To(cidr))
	}

	s := &armnetwork.Subnet{
		Name: ptr.To(cloud.Azure.SubnetName),
		Properties: &armnetwork.SubnetPropertiesFormat{
			RouteTable: &armnetwork.RouteTable{
				ID: routeTableID,
			},
		},
	}

	if len(cidrs) == 1 {
		s.Properties.AddressPrefix = ptr.To(cidrs[0])
	} else {
		s.Properties.AddressPrefixes = cidrPointers
	}

	return s
}

// ensureSubnet will create or update an Azure subnetwork in the specified vnet. The call is idempotent.
func ensureSubnet(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec, sn *armnetwork.Subnet) error {
	if sn == nil {
		return fmt.Errorf("invalid subnet reference")
	}

	resourceGroup := getResourceGroup(cloud)

	future, err := clients.Subnets.BeginCreateOrUpdate(ctx, resourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName, *sn, nil)
	if err != nil {
		return fmt.Errorf("failed to create or update subnetwork %q: %w", cloud.Azure.SubnetName, err)
	}

	_, err = future.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 5 * time.Second,
	})

	return err
}

func deleteSubnet(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	future, err := clients.Subnets.BeginDelete(ctx, getResourceGroup(cloud), cloud.Azure.VNetName, cloud.Azure.SubnetName, nil)
	if err != nil {
		return ignoreNotFound(err)
	}

	_, err = future.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 5 * time.Second,
	})

	return err
}
