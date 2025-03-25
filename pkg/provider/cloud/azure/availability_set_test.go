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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/utils/ptr"
)

func TestReconcileAvailabilitySet(t *testing.T) {
	credentials, err := getFakeCredentials()
	if err != nil {
		t.Fatalf("failed to generate credentials: %v", err)
	}

	testcases := []struct {
		name                        string
		clusterName                 string
		azureCloudSpec              *kubermaticv1.AzureCloudSpec
		existingAvailabilitySet     *armcompute.AvailabilitySet
		clientMode                  fakeClientMode
		overrideTags                bool
		overrideUpdateDomainCount   bool
		expectedError               bool
		expectedAvailabilitySetName string
		expectedFirstCallCount      int
		expectedSecondCallCount     int
	}{
		{
			name:                        "no-availability-set-name",
			clusterName:                 "94fs85s8mz",
			azureCloudSpec:              &kubermaticv1.AzureCloudSpec{},
			existingAvailabilitySet:     nil,
			clientMode:                  fakeClientModeOkay,
			overrideTags:                false,
			overrideUpdateDomainCount:   false,
			expectedError:               false,
			expectedAvailabilitySetName: "kubernetes-94fs85s8mz",
			expectedFirstCallCount:      1,
			expectedSecondCallCount:     1,
		},
		{
			name:                        "ownership-tag-removal",
			clusterName:                 "xxmhccmbx3",
			azureCloudSpec:              &kubermaticv1.AzureCloudSpec{},
			existingAvailabilitySet:     nil,
			clientMode:                  fakeClientModeOkay,
			overrideTags:                true,
			overrideUpdateDomainCount:   false,
			expectedError:               false,
			expectedAvailabilitySetName: "kubernetes-xxmhccmbx3",
			expectedFirstCallCount:      1,
			expectedSecondCallCount:     1,
		},
		{
			name:                        "change-update-domain-count",
			clusterName:                 "ku0of7owfh",
			azureCloudSpec:              &kubermaticv1.AzureCloudSpec{},
			existingAvailabilitySet:     nil,
			clientMode:                  fakeClientModeOkay,
			overrideTags:                false,
			overrideUpdateDomainCount:   true,
			expectedError:               false,
			expectedAvailabilitySetName: "kubernetes-ku0of7owfh",
			expectedFirstCallCount:      1,
			expectedSecondCallCount:     2,
		},
		{
			name:                        "take-over-resource-and-modify",
			clusterName:                 "kdh4pozvkw",
			azureCloudSpec:              &kubermaticv1.AzureCloudSpec{},
			existingAvailabilitySet:     nil,
			clientMode:                  fakeClientModeOkay,
			overrideTags:                true,
			overrideUpdateDomainCount:   true,
			expectedError:               false,
			expectedAvailabilitySetName: "kubernetes-kdh4pozvkw",
			expectedFirstCallCount:      1,
			expectedSecondCallCount:     1,
		},
		{
			name:        "custom-nonexistent-availability-set",
			clusterName: "x2ca7jkvgr",
			azureCloudSpec: &kubermaticv1.AzureCloudSpec{
				ResourceGroup:   customExistingResourceGroup,
				AvailabilitySet: customExistingAvailabilitySet,
			},
			existingAvailabilitySet:     nil,
			clientMode:                  fakeClientModeOkay,
			overrideTags:                false,
			overrideUpdateDomainCount:   false,
			expectedError:               false,
			expectedAvailabilitySetName: customExistingAvailabilitySet,
			expectedFirstCallCount:      1,
			expectedSecondCallCount:     1,
		},
		{
			name:        "existing-availability-set",
			clusterName: "n146b2u5h3",
			azureCloudSpec: &kubermaticv1.AzureCloudSpec{
				ResourceGroup:   customExistingResourceGroup,
				AvailabilitySet: customExistingAvailabilitySet,
			},
			existingAvailabilitySet: &armcompute.AvailabilitySet{
				ID:       ptr.To(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/%s", credentials.SubscriptionID, customExistingResourceGroup, customExistingAvailabilitySet)),
				Name:     ptr.To(customExistingAvailabilitySet),
				Location: ptr.To(testLocation),
				Type:     ptr.To("Microsoft.Compute/availabilitySets"),
			},
			clientMode:                  fakeClientModeOkay,
			overrideTags:                false,
			overrideUpdateDomainCount:   false,
			expectedError:               false,
			expectedAvailabilitySetName: customExistingAvailabilitySet,
			expectedFirstCallCount:      0,
			expectedSecondCallCount:     0,
		},
		{
			name:                        "invalid-credentials",
			clusterName:                 "m7t4oo7eai",
			azureCloudSpec:              &kubermaticv1.AzureCloudSpec{},
			existingAvailabilitySet:     nil,
			clientMode:                  fakeClientModeAuthFail,
			overrideTags:                false,
			overrideUpdateDomainCount:   false,
			expectedError:               true,
			expectedAvailabilitySetName: "",
			expectedFirstCallCount:      0,
			expectedSecondCallCount:     0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// prepare cluster resource and client set
			cluster := makeCluster(tc.clusterName, tc.azureCloudSpec, credentials)
			clientSet := getFakeClientSetWithAvailabilitySetsClient(tc.existingAvailabilitySet, tc.clientMode)

			fakeClient, ok := clientSet.AvailabilitySets.(*fakeAvailabilitySetsClient)
			if !ok {
				t.Fatalf("failed to access underlying fake AvailabilitySetsClient")
			}

			// reconcile AvailabilitySet the first time
			cluster, err = reconcileAvailabilitySet(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

			if tc.expectedError && err == nil {
				t.Fatal("expected first reconcileAvailabilitySet to fail, but succeeded without error")
			}

			if !tc.expectedError {
				if err != nil {
					t.Fatalf("expected first reconcileAvailabilitySet to succeed, but failed with error: %v", err)
				}

				if cluster.Spec.Cloud.Azure.AvailabilitySet != tc.expectedAvailabilitySetName {
					t.Fatalf("expected availability set in cloud spec to be '%s', got '%s'", tc.expectedAvailabilitySetName, cluster.Spec.Cloud.Azure.AvailabilitySet)
				}

				if fakeClient.CreateOrUpdateCalledCount != tc.expectedFirstCallCount {
					t.Fatalf("expected %d, got %d calls to CreateOrUpdate after first reconcile", tc.expectedFirstCallCount, fakeClient.CreateOrUpdateCalledCount)
				}

				if tc.overrideTags {
					// override all tags on the availability set
					fakeClient.AvailabilitySet.Tags = map[string]*string{}
				}

				// mess with the platform update domain count to force a reconcile
				if tc.overrideUpdateDomainCount {
					fakeClient.AvailabilitySet.Properties.PlatformUpdateDomainCount = ptr.To[int32](15)
				}

				// reconcile AvailabilitySet the second time
				cluster, err = reconcileAvailabilitySet(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

				if !tc.expectedError && err != nil {
					t.Fatalf("expected second reconcileAvailabilitySet to succeed, but failed with error: %v", err)
				}

				if cluster.Spec.Cloud.Azure.AvailabilitySet != tc.expectedAvailabilitySetName {
					t.Fatalf("expected availability set in cloud spec to be '%s', got '%s'", tc.expectedAvailabilitySetName, cluster.Spec.Cloud.Azure.AvailabilitySet)
				}

				if fakeClient.CreateOrUpdateCalledCount != tc.expectedSecondCallCount {
					t.Fatalf("expected %d, got %d calls to CreateOrUpdate after second reconcile", tc.expectedSecondCallCount, fakeClient.CreateOrUpdateCalledCount)
				}

				// make sure the reconcile fixed the update domain count if we didn't override ownership tags
				if tc.overrideUpdateDomainCount && !tc.overrideTags {
					if *fakeClient.AvailabilitySet.Properties.PlatformUpdateDomainCount != 20 {
						t.Fatalf("expected platform update domain count to be 20, got %d", *fakeClient.AvailabilitySet.Properties.PlatformUpdateDomainCount)
					}
				}
			}
		})
	}
}

const customExistingAvailabilitySet = "custom-existing-availability-set"

type fakeAvailabilitySetsClient struct {
	armcompute.AvailabilitySetsClient

	mode                      fakeClientMode
	AvailabilitySet           *armcompute.AvailabilitySet
	CreateOrUpdateCalledCount int
}

func getFakeClientSetWithAvailabilitySetsClient(existingAvailabilitySet *armcompute.AvailabilitySet, mode fakeClientMode) *ClientSet {
	return &ClientSet{
		AvailabilitySets: &fakeAvailabilitySetsClient{
			mode:            mode,
			AvailabilitySet: existingAvailabilitySet,
		},
	}
}

func (c *fakeAvailabilitySetsClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, availabilitySetName string, parameters armcompute.AvailabilitySet, options *armcompute.AvailabilitySetsClientCreateOrUpdateOptions) (armcompute.AvailabilitySetsClientCreateOrUpdateResponse, error) {
	c.CreateOrUpdateCalledCount++
	c.AvailabilitySet = &parameters

	return armcompute.AvailabilitySetsClientCreateOrUpdateResponse{
		AvailabilitySet: *c.AvailabilitySet,
	}, nil
}

func (c *fakeAvailabilitySetsClient) Get(ctx context.Context, resourceGroupName string, availabilitySetName string, options *armcompute.AvailabilitySetsClientGetOptions) (armcompute.AvailabilitySetsClientGetResponse, error) {
	switch c.mode {
	case fakeClientModeOkay:
		if c.AvailabilitySet != nil && c.AvailabilitySet.Name != nil && availabilitySetName == *c.AvailabilitySet.Name {
			return armcompute.AvailabilitySetsClientGetResponse{
				AvailabilitySet: *c.AvailabilitySet,
			}, nil
		}

		return armcompute.AvailabilitySetsClientGetResponse{}, &azcore.ResponseError{
			StatusCode: http.StatusNotFound,
		}

	case fakeClientModeAuthFail:
		return armcompute.AvailabilitySetsClientGetResponse{}, &azcore.ResponseError{
			StatusCode: http.StatusForbidden,
		}
	}

	return armcompute.AvailabilitySetsClientGetResponse{}, fmt.Errorf("unknown fake client mode: %s", c.mode)
}
