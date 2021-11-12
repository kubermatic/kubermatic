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

func subnetName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileSubnet(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	cluster.Spec.Cloud.Azure.SubnetName = subnetName(cluster)

	if err := ensureSubnet(ctx, clients, cluster.Spec.Cloud); err != nil {
		return cluster, err
	}

	return update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.SubnetName = cluster.Spec.Cloud.Azure.SubnetName
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerSubnet)
	})
}

// ensureSubnet will create or update an Azure subnetwork in the specified vnet. The call is idempotent.
func ensureSubnet(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	parameters := network.Subnet{
		Name: to.StringPtr(cloud.Azure.SubnetName),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr("10.0.0.0/16"),
		},
	}

	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}
	future, err := clients.Subnets.CreateOrUpdate(ctx, resourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName, parameters)
	if err != nil {
		return fmt.Errorf("failed to create or update subnetwork %q: %v", cloud.Azure.SubnetName, err)
	}

	if err = future.WaitForCompletionRef(ctx, *clients.Autorest); err != nil {
		return fmt.Errorf("failed to create or update subnetwork %q: %v", cloud.Azure.SubnetName, err)
	}

	return nil
}

func deleteSubnet(ctx context.Context, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	subnetsClient, err := getSubnetsClient(cloud, credentials)
	if err != nil {
		return err
	}

	var resourceGroup = cloud.Azure.ResourceGroup
	if cloud.Azure.VNetResourceGroup != "" {
		resourceGroup = cloud.Azure.VNetResourceGroup
	}
	deleteSubnetFuture, err := subnetsClient.Delete(ctx, resourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName)
	if err != nil {
		return err
	}

	if err = deleteSubnetFuture.WaitForCompletionRef(ctx, subnetsClient.Client); err != nil {
		return err
	}

	return nil
}
