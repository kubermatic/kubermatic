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

func routeTableName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileRouteTable(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if cluster.Spec.Cloud.Azure.RouteTableName == "" {
		cluster.Spec.Cloud.Azure.RouteTableName = routeTableName(cluster)
	}

	routeTable, err := clients.RouteTables.Get(ctx, cluster.Spec.Cloud.Azure.ResourceGroup, cluster.Spec.Cloud.Azure.RouteTableName, "")
	if err != nil && !isNotFound(routeTable.Response) {
		return cluster, err
	}

	// if we found a route table, we can check for the ownership tag to determine
	// if the referenced route table is owned by this cluster and should be reconciled
	if !isNotFound(routeTable.Response) && !hasOwnershipTag(routeTable.Tags, cluster) {
		return cluster, nil
	}

	target := targetRouteTable(cluster.Spec.Cloud, location)

	// check for attributes of the existing route table and return early if all values are already
	// as expected. Since there are a lot of pointers in the network.RouteTable struct, we need to
	// do a lot of "!= nil" checks so this does not panic.
	//
	// Attributes we check:
	// - Associated subnet's ID (subnet names are part of the ID and as such, don't need a separate check)
	if routeTable.RouteTablePropertiesFormat != nil && routeTable.RouteTablePropertiesFormat.Subnets != nil && len(*routeTable.RouteTablePropertiesFormat.Subnets) == 1 &&
		*(*routeTable.RouteTablePropertiesFormat.Subnets)[0].ID == *(*target.RouteTablePropertiesFormat.Subnets)[0].ID {
		return cluster, nil
	}

	if err := ensureRouteTable(ctx, clients, cluster.Spec.Cloud, target); err != nil {
		return cluster, err
	}

	return update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.RouteTableName = cluster.Spec.Cloud.Azure.RouteTableName
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerRouteTable)
	})
}

func targetRouteTable(cloud kubermaticv1.CloudSpec, location string) *network.RouteTable {
	return &network.RouteTable{
		Name:     to.StringPtr(cloud.Azure.RouteTableName),
		Location: to.StringPtr(location),
		RouteTablePropertiesFormat: &network.RouteTablePropertiesFormat{
			Subnets: &[]network.Subnet{
				{
					Name: to.StringPtr(cloud.Azure.SubnetName),
					ID:   to.StringPtr(assembleSubnetID(cloud)),
				},
			},
		},
	}
}

// ensureRouteTable will create or update an Azure route table attached to the specified subnet. The call is idempotent.
func ensureRouteTable(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec, rt *network.RouteTable) error {
	if rt == nil {
		return fmt.Errorf("invalid network.RouteTable passed")
	}

	future, err := clients.RouteTables.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName, *rt)
	if err != nil {
		return fmt.Errorf("failed to create or update route table %q: %v", cloud.Azure.RouteTableName, err)
	}

	if err = future.WaitForCompletionRef(ctx, *clients.Autorest); err != nil {
		return fmt.Errorf("failed to create or update route table %q: %v", cloud.Azure.RouteTableName, err)
	}

	return nil
}

func deleteRouteTable(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	future, err := clients.RouteTables.Delete(ctx, cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName)
	if err != nil {
		return err
	}

	if err = future.WaitForCompletionRef(ctx, *clients.Autorest); err != nil {
		return err
	}

	return nil
}
