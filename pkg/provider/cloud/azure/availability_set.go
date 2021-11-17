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

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-07-01/compute/computeapi"
	"github.com/Azure/go-autorest/autorest/to"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func availabilitySetName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileAvailabilitySet(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if cluster.Spec.Cloud.Azure.AvailabilitySet == "" {
		cluster.Spec.Cloud.Azure.AvailabilitySet = availabilitySetName(cluster)
	}

	availabilitySet, err := clients.AvailabilitySets.Get(ctx, cluster.Spec.Cloud.Azure.ResourceGroup, cluster.Spec.Cloud.Azure.AvailabilitySet)
	if err != nil && !isNotFound(availabilitySet.Response) {
		return cluster, err
	}

	// if we found an availability set, we can check for the ownership tag to determine
	// if the referenced availability set is owned by this cluster and should be reconciled
	if !isNotFound(availabilitySet.Response) && !hasOwnershipTag(availabilitySet.Tags, cluster) {
		return cluster, nil
	}

	if err := ensureAvailabilitySet(ctx, clients.AvailabilitySets, cluster.Spec.Cloud, location); err != nil {
		return nil, fmt.Errorf("failed to ensure AvailabilitySet exists: %v", err)
	}

	return update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.AvailabilitySet = cluster.Spec.Cloud.Azure.AvailabilitySet
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerAvailabilitySet)
	})
}

func ensureAvailabilitySet(ctx context.Context, client computeapi.AvailabilitySetsClientAPI, cloud kubermaticv1.CloudSpec, location string) error {
	faultDomainCount, ok := faultDomainsPerRegion[location]
	if !ok {
		return fmt.Errorf("could not determine the number of fault domains, unknown region %q", location)
	}

	as := compute.AvailabilitySet{
		Name:     to.StringPtr(cloud.Azure.AvailabilitySet),
		Location: to.StringPtr(location),
		Sku: &compute.Sku{
			Name: to.StringPtr("Aligned"),
		},
		AvailabilitySetProperties: &compute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  to.Int32Ptr(faultDomainCount),
			PlatformUpdateDomainCount: to.Int32Ptr(20),
		},
	}

	_, err := client.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, cloud.Azure.AvailabilitySet, as)
	if err != nil {
		return fmt.Errorf("failed to create or update availability set %q: %v", cloud.Azure.AvailabilitySet, err)
	}

	return nil
}

func deleteAvailabilitySet(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	_, err := clients.AvailabilitySets.Delete(ctx, cloud.Azure.ResourceGroup, cloud.Azure.AvailabilitySet)
	return err
}
