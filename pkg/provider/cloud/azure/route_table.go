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
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/ptr"
)

func routeTableName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileRouteTable(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	name := cluster.Spec.Cloud.Azure.RouteTableName

	if cluster.Spec.Cloud.Azure.RouteTableName == "" {
		cluster.Spec.Cloud.Azure.RouteTableName = routeTableName(cluster)
	}

	_, err := clients.RouteTables.Get(ctx, cluster.Spec.Cloud.Azure.ResourceGroup, cluster.Spec.Cloud.Azure.RouteTableName, nil)
	if err != nil && !isNotFound(err) {
		return nil, err
	}

	// if the request returned no error, it means the route table already exists and we can return early.
	// usually, we check for ownership tags here and then compare attributes of interest to a target representation
	// of the resource. Since there is nothing in the route table we could compare to eventually reconcile (the subnet setting
	// you see later on is ineffective), we skip all of that and return early if we found a route table during our API call earlier.
	if err == nil {
		return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.RouteTableName = cluster.Spec.Cloud.Azure.RouteTableName
			// this is a special case; because we cannot determine if a route table was created by
			// the controller or not, we only add the finalizer if by the beginning of this loop, the
			// name was not set. Otherwise we risk deleting a route table we do not own.
			if name == "" {
				kuberneteshelper.AddFinalizer(updatedCluster, FinalizerRouteTable)
			}
		})
	}

	target := targetRouteTable(cluster.Spec.Cloud, location)
	if err := ensureRouteTable(ctx, clients, cluster.Spec.Cloud, target); err != nil {
		return cluster, err
	}

	return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.RouteTableName = cluster.Spec.Cloud.Azure.RouteTableName
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerRouteTable)
	})
}

func targetRouteTable(cloud kubermaticv1.CloudSpec, location string) *armnetwork.RouteTable {
	return &armnetwork.RouteTable{
		Name:     ptr.To(cloud.Azure.RouteTableName),
		Location: ptr.To(location),
	}
}

// ensureRouteTable will create or update an Azure route table attached to the specified subnet. The call is idempotent.
func ensureRouteTable(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec, rt *armnetwork.RouteTable) error {
	if rt == nil {
		return fmt.Errorf("invalid network.RouteTable passed")
	}

	future, err := clients.RouteTables.BeginCreateOrUpdate(ctx, cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName, *rt, nil)
	if err != nil {
		return fmt.Errorf("failed to create or update route table %q: %w", cloud.Azure.RouteTableName, err)
	}

	_, err = future.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 5 * time.Second,
	})

	return err
}

func deleteRouteTable(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	future, err := clients.RouteTables.BeginDelete(ctx, cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName, nil)
	if err != nil {
		return ignoreNotFound(err)
	}

	_, err = future.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 5 * time.Second,
	})

	return err
}
