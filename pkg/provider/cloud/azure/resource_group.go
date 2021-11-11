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
	"github.com/Azure/go-autorest/autorest/to"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

// ensureResourceGroup will create or update an Azure resource group. The call is idempotent.
func ensureResourceGroup(ctx context.Context, cloud kubermaticv1.CloudSpec, location string, clusterName string, credentials Credentials) error {
	groupsClient, err := getGroupsClient(cloud, credentials)
	if err != nil {
		return err
	}

	parameters := resources.Group{
		Name:     to.StringPtr(cloud.Azure.ResourceGroup),
		Location: to.StringPtr(location),
		Tags: map[string]*string{
			clusterTagKey: to.StringPtr(clusterName),
		},
	}
	if _, err = groupsClient.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, parameters); err != nil {
		return fmt.Errorf("failed to create or update resource group %q: %v", cloud.Azure.ResourceGroup, err)
	}

	return nil
}

func deleteResourceGroup(ctx context.Context, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	groupsClient, err := getGroupsClient(cloud, credentials)
	if err != nil {
		return err
	}

	// We're doing a Get to see if its already gone or not.
	// We could also directly call delete but the error response would need to be unpacked twice to get the correct error message.
	// Doing a get is simpler.
	if _, err := groupsClient.Get(ctx, cloud.Azure.ResourceGroup); err != nil {
		return err
	}

	future, err := groupsClient.Delete(ctx, cloud.Azure.ResourceGroup)
	if err != nil {
		return err
	}

	if err = future.WaitForCompletionRef(ctx, groupsClient.Client); err != nil {
		return err
	}

	return nil
}
