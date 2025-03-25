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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func ignoreNotFound(err error) error {
	if isNotFound(err) {
		return nil
	}

	return err
}

func isNotFound(err error) bool {
	var aerr *azcore.ResponseError
	if err != nil && errors.As(err, &aerr) {
		return aerr.StatusCode == http.StatusNotFound
	}

	return false
}

func getResourceGroup(cloud kubermaticv1.CloudSpec) string {
	if cloud.Azure.VNetResourceGroup != "" {
		return cloud.Azure.VNetResourceGroup
	}

	return cloud.Azure.ResourceGroup
}

func hasOwnershipTag(tags map[string]*string, cluster *kubermaticv1.Cluster) bool {
	if value, ok := tags[clusterTagKey]; ok {
		return *value == cluster.Name
	}

	return false
}

func GetVMSize(ctx context.Context, credentials Credentials, location, vmName string) (*provider.NodeCapacity, error) {
	credential, err := credentials.ToAzureCredential()
	if err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	sizesClient, err := getSizesClient(credential, credentials.SubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for size client: %w", err)
	}

	// get all available VM size types for given location
	pager := sizesClient.NewListPager(location, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list sizes: %w", err)
		}

		for _, size := range page.Value {
			if strings.EqualFold(*size.Name, vmName) {
				capacity := provider.NewNodeCapacity()
				capacity.WithCPUCount(int(*size.NumberOfCores))

				if err := capacity.WithMemory(int(*size.MemoryInMB), "M"); err != nil {
					return nil, fmt.Errorf("error parsing machine memory: %w", err)
				}

				if err := capacity.WithStorage(int(*size.ResourceDiskSizeInMB), "M"); err != nil {
					return nil, fmt.Errorf("error parsing machine disk size: %w", err)
				}

				return capacity, nil
			}
		}
	}

	return nil, fmt.Errorf("could not find Azure VM size %q", vmName)
}
