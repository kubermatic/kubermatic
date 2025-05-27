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
	defaultVNetCIDRIPv4 = "10.0.0.0/16"
	defaultVNetCIDRIPv6 = "fd00::/48"
)

func vnetName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileVNet(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if cluster.Spec.Cloud.Azure.VNetName == "" {
		cluster.Spec.Cloud.Azure.VNetName = vnetName(cluster)
	}

	vnet, err := clients.Networks.Get(ctx, getResourceGroup(cluster.Spec.Cloud), cluster.Spec.Cloud.Azure.VNetName, nil)
	if err != nil && !isNotFound(err) {
		return cluster, err
	}

	// if we found a VNET (no error), we can check for the ownership tag to determine
	// if the referenced VNET is owned by this cluster and should be reconciled. We return
	// early if the vnet is not owned by us.
	if err == nil && !hasOwnershipTag(vnet.Tags, cluster) {
		return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.VNetName = cluster.Spec.Cloud.Azure.VNetName
		})
	}

	var cidrs []string
	if cluster.IsIPv4Only() || cluster.IsDualStack() {
		cidrs = append(cidrs, defaultVNetCIDRIPv4)
	}
	if cluster.IsIPv6Only() || cluster.IsDualStack() {
		cidrs = append(cidrs, defaultVNetCIDRIPv6)
	}
	target := targetVnet(cluster.Spec.Cloud, location, cluster.Name, cidrs)

	// check for attributes of the existing VNET and return early if all values are already
	// as expected. Since there are a lot of pointers in the network.VirtualNetwork struct, we need to
	// do a lot of "!= nil" checks so this does not panic.
	//
	// Attributes we check:
	// - Address space CIDR
	if vnet.Properties == nil || vnet.Properties.AddressSpace == nil || !reflect.DeepEqual(vnet.Properties.AddressSpace.AddressPrefixes, target.Properties.AddressSpace.AddressPrefixes) {
		if err := ensureVNet(ctx, clients, cluster.Spec.Cloud, target); err != nil {
			return nil, err
		}
	}

	return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.VNetName = cluster.Spec.Cloud.Azure.VNetName
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerVNet)
	})
}

func targetVnet(cloud kubermaticv1.CloudSpec, location string, clusterName string, cidrs []string) *armnetwork.VirtualNetwork {
	cidrPointers := []*string{}
	for _, cidr := range cidrs {
		cidrPointers = append(cidrPointers, ptr.To(cidr))
	}

	return &armnetwork.VirtualNetwork{
		Name:     ptr.To(cloud.Azure.VNetName),
		Location: ptr.To(location),
		Tags: map[string]*string{
			clusterTagKey: ptr.To(clusterName),
		},
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{AddressPrefixes: cidrPointers},
		},
	}
}

// ensureVNet will create or update an Azure virtual network in the specified resource group. The call is idempotent.
func ensureVNet(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec, vnet *armnetwork.VirtualNetwork) error {
	if vnet == nil {
		return fmt.Errorf("invalid vnet reference passed")
	}

	future, err := clients.Networks.BeginCreateOrUpdate(ctx, getResourceGroup(cloud), cloud.Azure.VNetName, *vnet, nil)
	if err != nil {
		return fmt.Errorf("failed to create or update virtual network %q: %w", cloud.Azure.VNetName, err)
	}

	_, err = future.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 5 * time.Second,
	})

	return err
}

func deleteVNet(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	future, err := clients.Networks.BeginDelete(ctx, getResourceGroup(cloud), cloud.Azure.VNetName, nil)
	if err != nil {
		return ignoreNotFound(err)
	}

	_, err = future.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 5 * time.Second,
	})

	return err
}
