//go:build integration

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
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

func TestReconcileResourceGroup(t *testing.T) {
	credentials, err := getFakeCredentials()
	if err != nil {
		t.Fatalf("failed to generate credentials: %v", err)
	}

	ctx := context.Background()

	t.Run("no-resource-group-set", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AzureCloudSpec{}, credentials)
		clientSet := getFakeClientSetWithGroupsClient(*credentials, testLocation, cluster, nil, fakeClientModeOkay)
		cluster, err = reconcileResourceGroup(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err != nil {
			t.Fatalf("expected ensureResourceGroup to succeed, but failed with error: %v", err)
		}

		if cluster.Spec.Cloud.Azure.ResourceGroup != resourceGroupName(cluster) {
			t.Fatalf("expected resource group in cloud spec to be '%s', got '%s'", resourceGroupName(cluster), cluster.Spec.Cloud.Azure.ResourceGroup)
		}
	})

	t.Run("ownership-tag-removal", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AzureCloudSpec{}, credentials)
		clientSet := getFakeClientSetWithGroupsClient(*credentials, testLocation, cluster, nil, fakeClientModeOkay)
		cluster, err = reconcileResourceGroup(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err != nil {
			t.Fatalf("expected ensureResourceGroup to succeed, but failed with error: %v", err)
		}

		if cluster.Spec.Cloud.Azure.ResourceGroup != resourceGroupName(cluster) {
			t.Fatalf("expected resource group in cloud spec to be '%s', got '%s'", resourceGroupName(cluster), cluster.Spec.Cloud.Azure.ResourceGroup)
		}

		fakeClient, ok := clientSet.Groups.(*fakeGroupsClient)
		if !ok {
			t.Fatalf("failed to access underlying fake GroupsClient")
		}

		if fakeClient.CreateOrUpdateCalledCount != 1 {
			t.Fatalf("expected update to resource group, got %d calls to CreateOrUpdate", fakeClient.CreateOrUpdateCalledCount)
		}

		// override all tags on the group
		fakeClient.Group.Tags = map[string]*string{}

		cluster, err = reconcileResourceGroup(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("expected ensureResourceGroup to succeed, but failed with error: %v", err)
		}

		if cluster.Spec.Cloud.Azure.ResourceGroup != resourceGroupName(cluster) {
			t.Fatalf("expected resource group in cloud spec to be '%s', got '%s'", resourceGroupName(cluster), cluster.Spec.Cloud.Azure.ResourceGroup)
		}

		if fakeClient.CreateOrUpdateCalledCount != 1 {
			t.Fatalf("expected no further update to resource group, got %d calls to CreateOrUpdate", fakeClient.CreateOrUpdateCalledCount)
		}

	})

	t.Run("custom-resource-group-exists", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AzureCloudSpec{
			ResourceGroup: customExistingResourceGroup,
		}, credentials)

		existingGroup := &resources.Group{
			ID:        to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", credentials.SubscriptionID, customExistingResourceGroup)),
			Name:      to.StringPtr(customExistingResourceGroup),
			Location:  to.StringPtr(testLocation),
			Type:      to.StringPtr("Microsoft.Resources/resourceGroups"),
			ManagedBy: nil,
			Properties: &resources.GroupProperties{
				ProvisioningState: to.StringPtr("Succeeded"),
			},
		}

		clientSet := getFakeClientSetWithGroupsClient(*credentials, testLocation, cluster, existingGroup, fakeClientModeOkay)
		cluster, err = reconcileResourceGroup(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err != nil {
			t.Fatalf("expected reconcileResourceGroup to succeed, but failed with error: %v", err)
		}

		if cluster.Spec.Cloud.Azure.ResourceGroup != customExistingResourceGroup {
			t.Fatalf("expected resource group in cloud spec to be '%s', got '%s'", customExistingResourceGroup, cluster.Spec.Cloud.Azure.ResourceGroup)
		}

		fakeClient, ok := clientSet.Groups.(*fakeGroupsClient)
		if !ok {
			t.Fatalf("failed to access underlying fake GroupsClient")
		}

		if fakeClient.CreateOrUpdateCalledCount != 0 {
			t.Fatalf("expected no attempts to update resource group, got %d calls to CreateOrUpdate", fakeClient.CreateOrUpdateCalledCount)
		}
	})

	t.Run("custom-resource-group-does-not-exist", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AzureCloudSpec{
			ResourceGroup: "does-not-exist",
		}, credentials)

		clientSet := getFakeClientSetWithGroupsClient(*credentials, testLocation, cluster, nil, fakeClientModeOkay)
		cluster, err = reconcileResourceGroup(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err != nil {
			t.Fatalf("expected reconcileResourceGroup to succeed, but failed with error: %v", err)
		}

		if cluster.Spec.Cloud.Azure.ResourceGroup != "does-not-exist" {
			t.Fatalf("expected resource group in cloud spec to be '%s', got '%s'", "does-not-exist", cluster.Spec.Cloud.Azure.ResourceGroup)
		}

		fakeClient, ok := clientSet.Groups.(*fakeGroupsClient)
		if !ok {
			t.Fatalf("failed to access underlying fake GroupsClient")
		}

		if fakeClient.CreateOrUpdateCalledCount != 1 {
			t.Fatalf("expected call to CreateOrUpdate, got %d calls", fakeClient.CreateOrUpdateCalledCount)
		}
	})

	t.Run("invalid-credentials", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AzureCloudSpec{}, credentials)

		clientSet := getFakeClientSetWithGroupsClient(*credentials, testLocation, cluster, nil, fakeClientModeAuthFail)
		_, err := reconcileResourceGroup(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err == nil {
			t.Fatalf("expected error for request that got a 403 error, got none")
		}
	})
}

const customExistingResourceGroup = "custom-existing-resource-group"

type fakeGroupsClient struct {
	resources.GroupsClient
	location    string
	credentials Credentials
	mode        fakeClientMode
	cluster     *kubermaticv1.Cluster

	Group *resources.Group

	CreateOrUpdateCalledCount int
}

func getFakeClientSetWithGroupsClient(credentials Credentials, location string, cluster *kubermaticv1.Cluster, existingGroup *resources.Group, mode fakeClientMode) *ClientSet {
	return &ClientSet{
		Groups: &fakeGroupsClient{
			location:    location,
			credentials: credentials,
			mode:        mode,
			cluster:     cluster,
			Group:       existingGroup,
		},
	}
}

func (c *fakeGroupsClient) Get(ctx context.Context, groupName string) (result resources.Group, err error) {
	switch c.mode {
	case fakeClientModeOkay:
		if c.Group != nil && c.Group.Name != nil && groupName == *c.Group.Name {
			return *c.Group, nil
		} else {
			resp := autorest.Response{
				Response: &http.Response{
					StatusCode: 404,
				},
			}

			return resources.Group{
				Response: resp,
			}, autorest.NewErrorWithError(fmt.Errorf("not found"), "resources.GroupsClient", "Get", resp.Response, "Failure responding to request")
		}

	case fakeClientModeAuthFail:
		resp := autorest.Response{
			Response: &http.Response{
				StatusCode: 403,
			},
		}

		return resources.Group{
			Response: resp,
		}, autorest.NewErrorWithError(fmt.Errorf("unauthorized"), "resources.GroupsClient", "Get", resp.Response, "Failure responding to request")
	}

	return resources.Group{}, fmt.Errorf("unknown fake client mode: %s", c.mode)
}

func (c *fakeGroupsClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, parameters resources.Group) (result resources.Group, err error) {
	c.CreateOrUpdateCalledCount++
	c.Group = &parameters

	return *c.Group, nil
}
