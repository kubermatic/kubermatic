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

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func subnetName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileSubnet(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if cluster.Spec.Cloud.Azure.SubnetName == "" {
		cluster.Spec.Cloud.Azure.SubnetName = subnetName(cluster)
	}

	resourceGroup := cluster.Spec.Cloud.Azure.ResourceGroup
	if cluster.Spec.Cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cluster.Spec.Cloud.Azure.VNetResourceGroup
	}

	vnet, err := clients.Networks.Get(ctx, resourceGroup, cluster.Spec.Cloud.Azure.VNetName, "")
	if err != nil && !isNotFound(vnet.Response) {
		return cluster, err
	}

	subnet, err := clients.Subnets.Get(ctx, resourceGroup, *vnet.Name, cluster.Spec.Cloud.Azure.SubnetName, "")
	if err != nil && !isNotFound(subnet.Response) {
		return nil, err
	}

	// since subnets are sub-resources of VNETs and don't have tags themselves
	// we can only guess KKP ownership based on the VNET ownership tag. If the
	// VNET isn't owned by KKP, we should not try to reconcile subnets and
	// return early
	if !isNotFound(subnet.Response) && !hasOwnershipTag(vnet.Tags, cluster) {
		return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.SubnetName = cluster.Spec.Cloud.Azure.SubnetName
		})
	}

	target := targetSubnet(cluster.Spec.Cloud)

	// check for attributes of the existing subnet and skip ensuring if all values are already
	// as expected. Since there are a lot of pointers in the network.Subnet struct, we need to
	// do a lot of "!= nil" checks so this does not panic.
	//
	// Attributes we check:
	// - Subnet CIDR
	if !(subnet.SubnetPropertiesFormat != nil && subnet.SubnetPropertiesFormat.AddressPrefix != nil && *subnet.SubnetPropertiesFormat.AddressPrefix == *target.SubnetPropertiesFormat.AddressPrefix) {
		if err := ensureSubnet(ctx, clients, cluster.Spec.Cloud, target); err != nil {
			return nil, err
		}
	}

	return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.SubnetName = cluster.Spec.Cloud.Azure.SubnetName
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerSubnet)
	})
}

func targetSubnet(cloud kubermaticv1.CloudSpec) *network.Subnet {
	return &network.Subnet{
		Name: to.StringPtr(cloud.Azure.SubnetName),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr("10.0.0.0/16"),
		},
	}
}

// ensureSubnet will create or update an Azure subnetwork in the specified vnet. The call is idempotent.
func ensureSubnet(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec, sn *network.Subnet) error {
	if sn == nil {
		return fmt.Errorf("invalid subnet reference")
	}

	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}
	future, err := clients.Subnets.CreateOrUpdate(ctx, resourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName, *sn)
	if err != nil {
		return fmt.Errorf("failed to create or update subnetwork %q: %w", cloud.Azure.SubnetName, err)
	}

	if err = future.WaitForCompletionRef(ctx, *clients.Autorest); err != nil {
		return fmt.Errorf("failed to create or update subnetwork %q: %w", cloud.Azure.SubnetName, err)
	}

	return nil
}

func deleteSubnet(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}
	deleteSubnetFuture, err := clients.Subnets.Delete(ctx, resourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName)
	if err != nil {
		return err
	}

	if err = deleteSubnetFuture.WaitForCompletionRef(ctx, *clients.Autorest); err != nil {
		return err
	}

	return nil
}
