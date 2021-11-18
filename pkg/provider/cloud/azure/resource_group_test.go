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
		clientSet := getFakeClientSetWithGroupsClient(*credentials, testLocation, cluster, fakeClientModeOkay)
		cluster, err = reconcileResourceGroup(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err != nil {
			t.Fatalf("expected ensureResourceGroup to succeed, but failed with error: %v", err)
		}

		if cluster.Spec.Cloud.Azure.ResourceGroup != resourceGroupName(cluster) {
			t.Fatalf("expected resource group in cloud spec to be '%s', got '%s'", resourceGroupName(cluster), cluster.Spec.Cloud.Azure.ResourceGroup)
		}
	})

	t.Run("custom-resource-group-exists", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AzureCloudSpec{
			ResourceGroup: customExistingResourceGroup,
		}, credentials)
		clientSet := getFakeClientSetWithGroupsClient(*credentials, testLocation, cluster, fakeClientModeOkay)
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

		clientSet := getFakeClientSetWithGroupsClient(*credentials, testLocation, cluster, fakeClientModeOkay)
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

		clientSet := getFakeClientSetWithGroupsClient(*credentials, testLocation, cluster, fakeClientModeAuthFail)
		_, err := reconcileResourceGroup(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err == nil {
			t.Fatalf("expected error for request that got a 403 error, got none")
		}
	})
}

const customExistingResourceGroup = "custom-existing-resource-groupe"

type fakeGroupsClient struct {
	resources.GroupsClient
	location    string
	credentials Credentials
	mode        fakeClientMode
	cluster     *kubermaticv1.Cluster

	CreateOrUpdateCalledCount int
	CreateOrUpdateGroup       resources.Group
}

func getFakeClientSetWithGroupsClient(credentials Credentials, location string, cluster *kubermaticv1.Cluster, mode fakeClientMode) *ClientSet {
	return &ClientSet{
		Groups: &fakeGroupsClient{
			location:    location,
			credentials: credentials,
			mode:        mode,
			cluster:     cluster,
		},
	}
}

func (c *fakeGroupsClient) Get(ctx context.Context, groupName string) (result resources.Group, err error) {
	switch c.mode {
	case fakeClientModeOkay:
		if groupName == customExistingResourceGroup {
			return resources.Group{
				ID:        to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.credentials.SubscriptionID, customExistingResourceGroup)),
				Name:      to.StringPtr(customExistingResourceGroup),
				Location:  &c.location,
				Type:      to.StringPtr("Microsoft.Resources/resourceGroups"),
				ManagedBy: nil,
				Properties: &resources.GroupProperties{
					ProvisioningState: to.StringPtr("Succeeded"),
				},
			}, nil
		} else if groupName == resourceGroupName(c.cluster) {
			return resources.Group{
				ID:        to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", c.credentials.SubscriptionID, resourceGroupName(c.cluster))),
				Name:      to.StringPtr(securityGroupName(c.cluster)),
				Location:  &c.location,
				Type:      to.StringPtr("Microsoft.Resources/resourceGroups"),
				ManagedBy: nil,
				Properties: &resources.GroupProperties{
					ProvisioningState: to.StringPtr("Succeeded"),
				},
				Tags: map[string]*string{
					clusterTagKey: to.StringPtr(c.cluster.ClusterName),
				},
			}, nil
		}

		resp := autorest.Response{
			Response: &http.Response{
				StatusCode: 404,
			},
		}

		return resources.Group{
			Response: resp,
		}, autorest.NewErrorWithError(fmt.Errorf("not found"), "resources.GroupsClient", "Get", resp.Response, "Failure responding to request")

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
	c.CreateOrUpdateGroup = parameters

	return parameters, nil
}
