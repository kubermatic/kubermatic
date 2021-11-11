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
	"github.com/Azure/go-autorest/autorest/to"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

func availabilitySetName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func ensureAvailabilitySet(ctx context.Context, name, location string, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	client, err := getAvailabilitySetClient(cloud, credentials)
	if err != nil {
		return err
	}

	faultDomainCount, ok := faultDomainsPerRegion[location]
	if !ok {
		return fmt.Errorf("could not determine the number of fault domains, unknown region %q", location)
	}

	as := compute.AvailabilitySet{
		Name:     to.StringPtr(name),
		Location: to.StringPtr(location),
		Sku: &compute.Sku{
			Name: to.StringPtr("Aligned"),
		},
		AvailabilitySetProperties: &compute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  to.Int32Ptr(faultDomainCount),
			PlatformUpdateDomainCount: to.Int32Ptr(20),
		},
	}

	_, err = client.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, name, as)
	return err
}

func deleteAvailabilitySet(ctx context.Context, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	asClient, err := getAvailabilitySetClient(cloud, credentials)
	if err != nil {
		return err
	}

	_, err = asClient.Delete(ctx, cloud.Azure.ResourceGroup, cloud.Azure.AvailabilitySet)
	return err
}
