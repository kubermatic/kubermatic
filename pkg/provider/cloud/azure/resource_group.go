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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/ptr"
)

func resourceGroupName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileResourceGroup(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	name := cluster.Spec.Cloud.Azure.ResourceGroup

	if cluster.Spec.Cloud.Azure.ResourceGroup == "" {
		cluster.Spec.Cloud.Azure.ResourceGroup = resourceGroupName(cluster)
	}

	_, err := clients.Groups.Get(ctx, cluster.Spec.Cloud.Azure.ResourceGroup, nil)
	if err != nil && !isNotFound(err) {
		return nil, err
	}

	// if the request returned no error, it means the resource group already exists and we can return early.
	// usually, we check for ownership tags here and then compare attributes of interest to a target representation
	// of the resource. Since there is nothing in the resource group we could compare to eventually reconcile, we
	// skip all of that and return early if we found a resource group during our API call earlier.
	if err == nil {
		return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.ResourceGroup = cluster.Spec.Cloud.Azure.ResourceGroup
			// this is a special case; because we cannot determine if a resource group was created by
			// the controller or not, we only add the finalizer if by the beginning of this loop, the
			// name was not set. Otherwise we risk deleting a resource group we do not own.
			if name == "" {
				kuberneteshelper.AddFinalizer(updatedCluster, FinalizerResourceGroup)
			}
		})
	}

	if err = ensureResourceGroup(ctx, clients.Groups, cluster.Spec.Cloud, location, cluster.Name); err != nil {
		return nil, err
	}

	return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.ResourceGroup = cluster.Spec.Cloud.Azure.ResourceGroup
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerResourceGroup)
	})
}

// ensureResourceGroup will create or update an Azure resource group. The call is idempotent.
func ensureResourceGroup(ctx context.Context, groupsClient ResourceGroupClient, cloud kubermaticv1.CloudSpec, location string, clusterName string) error {
	parameters := armresources.ResourceGroup{
		Name:     ptr.To(cloud.Azure.ResourceGroup),
		Location: ptr.To(location),
		Tags: map[string]*string{
			clusterTagKey: ptr.To(clusterName),
		},
	}
	if _, err := groupsClient.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, parameters, nil); err != nil {
		return fmt.Errorf("failed to create or update resource group %q: %w", cloud.Azure.ResourceGroup, err)
	}

	return nil
}

func deleteResourceGroup(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	future, err := clients.Groups.BeginDelete(ctx, cloud.Azure.ResourceGroup, nil)
	if err != nil {
		return ignoreNotFound(err)
	}

	_, err = future.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 5 * time.Second,
	})

	return err
}
