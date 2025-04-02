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

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/ptr"
)

func availabilitySetName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileAvailabilitySet(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if cluster.Spec.Cloud.Azure.AvailabilitySet == "" {
		cluster.Spec.Cloud.Azure.AvailabilitySet = availabilitySetName(cluster)
	}

	availabilitySet, err := clients.AvailabilitySets.Get(ctx, cluster.Spec.Cloud.Azure.ResourceGroup, cluster.Spec.Cloud.Azure.AvailabilitySet, nil)
	if err != nil && !isNotFound(err) {
		return nil, err
	}

	// if we found an availability set (no error), we can check for the ownership tag to determine
	// if the referenced availability set is owned by this cluster and should be reconciled. We return
	// early if the availability set is not owned by us.
	if err == nil && !hasOwnershipTag(availabilitySet.Tags, cluster) {
		return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.AvailabilitySet = cluster.Spec.Cloud.Azure.AvailabilitySet
		})
	}

	target, err := targetAvailabilitySet(cluster.Spec.Cloud, location, cluster.Name)
	if err != nil {
		return nil, err
	}

	// check for attributes of the existing availability set and return early if all values are already
	// as expected. Since there are a lot of pointers in the compute.AvailabilitySet struct, we need to
	// do a lot of "!= nil" checks so this does not panic.
	//
	// Attributes we check:
	// - SKU name
	// - fault domain count
	// - update domain count
	if availabilitySet.SKU == nil || availabilitySet.SKU.Name == nil || *availabilitySet.SKU.Name != *target.SKU.Name ||
		availabilitySet.Properties == nil ||
		availabilitySet.Properties.PlatformFaultDomainCount == nil || *availabilitySet.Properties.PlatformFaultDomainCount != *target.Properties.PlatformFaultDomainCount ||
		availabilitySet.Properties.PlatformUpdateDomainCount == nil || *availabilitySet.Properties.PlatformUpdateDomainCount != *target.Properties.PlatformUpdateDomainCount {
		if err := ensureAvailabilitySet(ctx, clients.AvailabilitySets, cluster.Spec.Cloud, target); err != nil {
			return nil, fmt.Errorf("failed to ensure AvailabilitySet exists: %w", err)
		}
	}

	return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.AvailabilitySet = cluster.Spec.Cloud.Azure.AvailabilitySet
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerAvailabilitySet)
	})
}

func targetAvailabilitySet(cloud kubermaticv1.CloudSpec, location string, clusterName string) (*armcompute.AvailabilitySet, error) {
	faultDomainCount, err := getRegionFaultDomainCount(location)
	if err != nil {
		return nil, fmt.Errorf("failed to get region fault domain count: %w", err)
	}

	return &armcompute.AvailabilitySet{
		Name:     ptr.To(cloud.Azure.AvailabilitySet),
		Location: ptr.To(location),
		SKU: &armcompute.SKU{
			Name: ptr.To("Aligned"),
		},
		Tags: map[string]*string{
			clusterTagKey: ptr.To(clusterName),
		},
		Properties: &armcompute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  ptr.To[int32](faultDomainCount),
			PlatformUpdateDomainCount: ptr.To[int32](20),
		},
	}, nil
}

func ensureAvailabilitySet(ctx context.Context, client AvailabilitySetClient, cloud kubermaticv1.CloudSpec, as *armcompute.AvailabilitySet) error {
	if as == nil {
		return fmt.Errorf("invalid AvailabilitySet reference passed, cannot be nil")
	}

	_, err := client.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, cloud.Azure.AvailabilitySet, *as, nil)
	if err != nil {
		return fmt.Errorf("failed to create or update availability set %q: %w", cloud.Azure.AvailabilitySet, err)
	}

	return nil
}

func deleteAvailabilitySet(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	_, err := clients.AvailabilitySets.Delete(ctx, cloud.Azure.ResourceGroup, cloud.Azure.AvailabilitySet, nil)
	return ignoreNotFound(err)
}
