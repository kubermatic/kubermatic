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

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func vnetName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileVNet(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	cluster.Spec.Cloud.Azure.VNetName = vnetName(cluster)

	if err := ensureVNet(ctx, clients, cluster.Spec.Cloud, location, cluster.Name); err != nil {
		return cluster, err
	}

	return update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.VNetName = cluster.Spec.Cloud.Azure.VNetName
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerVNet)
	})
}

// ensureVNet will create or update an Azure virtual network in the specified resource group. The call is idempotent.
func ensureVNet(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec, location string, clusterName string) error {
	parameters := network.VirtualNetwork{
		Name:     to.StringPtr(cloud.Azure.VNetName),
		Location: to.StringPtr(location),
		Tags: map[string]*string{
			clusterTagKey: to.StringPtr(clusterName),
		},
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{AddressPrefixes: &[]string{"10.0.0.0/16"}},
		},
	}

	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}
	future, err := clients.Networks.CreateOrUpdate(ctx, resourceGroup, cloud.Azure.VNetName, parameters)
	if err != nil {
		return fmt.Errorf("failed to create or update virtual network %q: %v", cloud.Azure.VNetName, err)
	}

	if err = future.WaitForCompletionRef(ctx, *clients.Autorest); err != nil {
		return fmt.Errorf("failed to create or update virtual network %q: %v", cloud.Azure.VNetName, err)
	}

	return nil
}

func deleteVNet(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}
	deleteVNetFuture, err := clients.Networks.Delete(ctx, resourceGroup, cloud.Azure.VNetName)
	if err != nil {
		return err
	}

	if err = deleteVNetFuture.WaitForCompletionRef(ctx, *clients.Autorest); err != nil {
		return err
	}

	return nil
}
