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

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources/resourcesapi"
	"github.com/Azure/go-autorest/autorest/to"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func resourceGroupName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileResourceGroup(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	name := cluster.Spec.Cloud.Azure.ResourceGroup

	if cluster.Spec.Cloud.Azure.ResourceGroup == "" {
		cluster.Spec.Cloud.Azure.ResourceGroup = resourceGroupName(cluster)
	}

	resourceGroup, err := clients.Groups.Get(ctx, cluster.Spec.Cloud.Azure.ResourceGroup)
	if err != nil && !isNotFound(resourceGroup.Response) {
		return nil, err
	}

	// usually, we check for ownership tags here and then compare attributes of interest to a target representation
	// of the resource. Since there is nothing in the resource group we could compare to eventually reconcile, we
	// skip all of that and return early if we found a resource group during our API call earlier.
	if !isNotFound(resourceGroup.Response) {
		return update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
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

	return update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.ResourceGroup = cluster.Spec.Cloud.Azure.ResourceGroup
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerResourceGroup)
	})
}

// ensureResourceGroup will create or update an Azure resource group. The call is idempotent.
func ensureResourceGroup(ctx context.Context, groupsClient resourcesapi.GroupsClientAPI, cloud kubermaticv1.CloudSpec, location string, clusterName string) error {
	parameters := resources.Group{
		Name:     to.StringPtr(cloud.Azure.ResourceGroup),
		Location: to.StringPtr(location),
		Tags: map[string]*string{
			clusterTagKey: to.StringPtr(clusterName),
		},
	}
	if _, err := groupsClient.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, parameters); err != nil {
		return fmt.Errorf("failed to create or update resource group %q: %w", cloud.Azure.ResourceGroup, err)
	}

	return nil
}

func deleteResourceGroup(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	// We're doing a Get to see if its already gone or not.
	// We could also directly call delete but the error response would need to be unpacked twice to get the correct error message.
	// Doing a get is simpler.
	if _, err := clients.Groups.Get(ctx, cloud.Azure.ResourceGroup); err != nil {
		return err
	}

	future, err := clients.Groups.Delete(ctx, cloud.Azure.ResourceGroup)
	if err != nil {
		return err
	}

	if err = future.WaitForCompletionRef(ctx, *clients.Autorest); err != nil {
		return err
	}

	return nil
}
