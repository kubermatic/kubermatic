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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/utils/ptr"
)

func TestReconcileResourceGroup(t *testing.T) {
	credentials, err := getFakeCredentials()
	if err != nil {
		t.Fatalf("failed to generate credentials: %v", err)
	}

	testcases := []struct {
		name                      string
		clusterName               string
		azureCloudSpec            *kubermaticv1.AzureCloudSpec
		existingResourceGroup     *armresources.ResourceGroup
		clientMode                fakeClientMode
		overrideTags              bool
		expectedError             bool
		expectedResourceGroupName string
		expectedFirstCallCount    int
		expectedSecondCallCount   int
	}{
		{
			name:                      "no-resource-group-name",
			clusterName:               "0stw4oaurg",
			azureCloudSpec:            &kubermaticv1.AzureCloudSpec{},
			existingResourceGroup:     nil,
			clientMode:                fakeClientModeOkay,
			overrideTags:              false,
			expectedError:             false,
			expectedResourceGroupName: "kubernetes-0stw4oaurg",
			expectedFirstCallCount:    1,
			expectedSecondCallCount:   1,
		},
		{
			name:                      "ownership-tag-removal",
			clusterName:               "aeed12dy19",
			azureCloudSpec:            &kubermaticv1.AzureCloudSpec{},
			existingResourceGroup:     nil,
			clientMode:                fakeClientModeOkay,
			overrideTags:              true,
			expectedError:             false,
			expectedResourceGroupName: "kubernetes-aeed12dy19",
			expectedFirstCallCount:    1,
			expectedSecondCallCount:   1,
		},
		{
			name:        "custom-nonexistent-resource-group",
			clusterName: "rt81kw1vhp",
			azureCloudSpec: &kubermaticv1.AzureCloudSpec{
				ResourceGroup: customExistingResourceGroup,
			},
			existingResourceGroup:     nil,
			clientMode:                fakeClientModeOkay,
			overrideTags:              false,
			expectedError:             false,
			expectedResourceGroupName: customExistingResourceGroup,
			expectedFirstCallCount:    1,
			expectedSecondCallCount:   1,
		},
		{
			name:        "existing-resource-group",
			clusterName: "knkfqapvsg",
			azureCloudSpec: &kubermaticv1.AzureCloudSpec{
				ResourceGroup: customExistingResourceGroup,
			},
			existingResourceGroup: &armresources.ResourceGroup{
				ID:        ptr.To(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", credentials.SubscriptionID, customExistingResourceGroup)),
				Name:      ptr.To(customExistingResourceGroup),
				Location:  ptr.To(testLocation),
				Type:      ptr.To("Microsoft.Resources/resourceGroups"),
				ManagedBy: nil,
				Properties: &armresources.ResourceGroupProperties{
					ProvisioningState: ptr.To("Succeeded"),
				},
			},
			clientMode:                fakeClientModeOkay,
			overrideTags:              false,
			expectedError:             false,
			expectedResourceGroupName: customExistingResourceGroup,
			expectedFirstCallCount:    0,
			expectedSecondCallCount:   0,
		},
		{
			name:                      "invalid-credentials",
			clusterName:               "1pft80obi4",
			azureCloudSpec:            &kubermaticv1.AzureCloudSpec{},
			existingResourceGroup:     nil,
			clientMode:                fakeClientModeAuthFail,
			overrideTags:              false,
			expectedError:             true,
			expectedResourceGroupName: "",
			expectedFirstCallCount:    0,
			expectedSecondCallCount:   0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// prepare cluster resource and client set
			cluster := makeCluster(tc.clusterName, tc.azureCloudSpec, credentials)
			clientSet := getFakeClientSetWithGroupsClient(tc.existingResourceGroup, tc.clientMode)

			fakeClient, ok := clientSet.Groups.(*fakeGroupsClient)
			if !ok {
				t.Fatalf("failed to access underlying fake GroupsClient")
			}

			// reconcile resource group the first time
			cluster, err = reconcileResourceGroup(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

			if tc.expectedError && err == nil {
				t.Fatal("expected first reconcileResourceGroup to fail, but succeeded without error")
			}

			if !tc.expectedError {
				if err != nil {
					t.Fatalf("expected first reconcileResourceGroup to succeed, but failed with error: %v", err)
				}

				if cluster.Spec.Cloud.Azure.ResourceGroup != tc.expectedResourceGroupName {
					t.Fatalf("expected resource group in cloud spec to be '%s', got '%s'", tc.expectedResourceGroupName, cluster.Spec.Cloud.Azure.ResourceGroup)
				}

				if fakeClient.CreateOrUpdateCalledCount != tc.expectedFirstCallCount {
					t.Fatalf("expected %d, got %d calls to CreateOrUpdate after first reconcile", tc.expectedFirstCallCount, fakeClient.CreateOrUpdateCalledCount)
				}

				if tc.overrideTags {
					// override all tags on the availability set
					fakeClient.Group.Tags = map[string]*string{}
				}

				// reconcile ResourceGroup the second time
				cluster, err = reconcileResourceGroup(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

				if !tc.expectedError && err != nil {
					t.Fatalf("expected second reconcileResourceGroup to succeed, but failed with error: %v", err)
				}

				if cluster.Spec.Cloud.Azure.ResourceGroup != tc.expectedResourceGroupName {
					t.Fatalf("expected resource group in cloud spec to be '%s', got '%s'", tc.expectedResourceGroupName, cluster.Spec.Cloud.Azure.ResourceGroup)
				}

				if fakeClient.CreateOrUpdateCalledCount != tc.expectedSecondCallCount {
					t.Fatalf("expected %d, got %d calls to CreateOrUpdate after second reconcile", tc.expectedSecondCallCount, fakeClient.CreateOrUpdateCalledCount)
				}
			}
		})
	}
}

const customExistingResourceGroup = "custom-existing-resource-group"

type fakeGroupsClient struct {
	armresources.ResourceGroupsClient

	mode                      fakeClientMode
	Group                     *armresources.ResourceGroup
	CreateOrUpdateCalledCount int
}

func getFakeClientSetWithGroupsClient(existingGroup *armresources.ResourceGroup, mode fakeClientMode) *ClientSet {
	return &ClientSet{
		Groups: &fakeGroupsClient{
			mode:  mode,
			Group: existingGroup,
		},
	}
}

func (c *fakeGroupsClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, parameters armresources.ResourceGroup, options *armresources.ResourceGroupsClientCreateOrUpdateOptions) (armresources.ResourceGroupsClientCreateOrUpdateResponse, error) {
	c.CreateOrUpdateCalledCount++
	c.Group = &parameters

	return armresources.ResourceGroupsClientCreateOrUpdateResponse{
		ResourceGroup: *c.Group,
	}, nil
}

func (c *fakeGroupsClient) Get(ctx context.Context, resourceGroupName string, options *armresources.ResourceGroupsClientGetOptions) (armresources.ResourceGroupsClientGetResponse, error) {
	switch c.mode {
	case fakeClientModeOkay:
		if c.Group != nil && c.Group.Name != nil && resourceGroupName == *c.Group.Name {
			return armresources.ResourceGroupsClientGetResponse{
				ResourceGroup: *c.Group,
			}, nil
		}

		return armresources.ResourceGroupsClientGetResponse{}, &azcore.ResponseError{
			StatusCode: http.StatusNotFound,
		}

	case fakeClientModeAuthFail:
		return armresources.ResourceGroupsClientGetResponse{}, &azcore.ResponseError{
			StatusCode: http.StatusForbidden,
		}
	}

	return armresources.ResourceGroupsClientGetResponse{}, fmt.Errorf("unknown fake client mode: %s", c.mode)
}
