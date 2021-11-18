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

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-07-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

func TestReconcileAvailabilitySet(t *testing.T) {
	credentials, err := getFakeCredentials()
	if err != nil {
		t.Fatalf("failed to generate credentials: %v", err)
	}

	ctx := context.Background()

	t.Run("no-availability-set-set", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AzureCloudSpec{}, credentials)
		clientSet := getFakeClientSetWithAvailabilitySetsClient(*credentials, testLocation, cluster, nil, fakeClientModeOkay)
		cluster, err = reconcileAvailabilitySet(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err != nil {
			t.Fatalf("expected reconcileAvailabilitySet to succeed, but failed with error: %v", err)
		}

		if cluster.Spec.Cloud.Azure.AvailabilitySet != availabilitySetName(cluster) {
			t.Fatalf("expected availability set in cloud spec to be '%s', got '%s'", availabilitySetName(cluster), cluster.Spec.Cloud.Azure.AvailabilitySet)
		}
	})

	t.Run("ownership-tag-removal", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AzureCloudSpec{}, credentials)
		clientSet := getFakeClientSetWithAvailabilitySetsClient(*credentials, testLocation, cluster, nil, fakeClientModeOkay)
		cluster, err = reconcileAvailabilitySet(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err != nil {
			t.Fatalf("expected reconcileAvailabilitySet to succeed, but failed with error: %v", err)
		}

		if cluster.Spec.Cloud.Azure.AvailabilitySet != availabilitySetName(cluster) {
			t.Fatalf("expected availability set in cloud spec to be '%s', got '%s'", availabilitySetName(cluster), cluster.Spec.Cloud.Azure.AvailabilitySet)
		}

		fakeClient, ok := clientSet.AvailabilitySets.(*fakeAvailabilitySetsClient)
		if !ok {
			t.Fatalf("failed to access underlying fake AvailabilitySetsClient")
		}

		if fakeClient.CreateOrUpdateCalledCount != 1 {
			t.Fatalf("expected update to availability set, got %d calls to CreateOrUpdate", fakeClient.CreateOrUpdateCalledCount)
		}

		// override all tags on the availability set
		fakeClient.AvailabilitySet.Tags = map[string]*string{}

		cluster, err = reconcileAvailabilitySet(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))
		if err != nil {
			t.Fatalf("expected reconcileAvailabilitySet to succeed, but failed with error: %v", err)
		}

		if cluster.Spec.Cloud.Azure.AvailabilitySet != availabilitySetName(cluster) {
			t.Fatalf("expected availability set in cloud spec to be '%s', got '%s'", availabilitySetName(cluster), cluster.Spec.Cloud.Azure.AvailabilitySet)
		}

		if fakeClient.CreateOrUpdateCalledCount != 1 {
			t.Fatalf("expected no further update to availability set, got %d calls to CreateOrUpdate", fakeClient.CreateOrUpdateCalledCount)
		}
	})

	t.Run("custom-availability-set-exists", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AzureCloudSpec{
			ResourceGroup:   customExistingResourceGroup,
			AvailabilitySet: customExistingAvailabilitySet,
		}, credentials)

		existingAvailabilitySet := &compute.AvailabilitySet{
			ID:       to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", credentials.SubscriptionID, customExistingResourceGroup)),
			Name:     to.StringPtr(customExistingAvailabilitySet),
			Location: to.StringPtr(testLocation),
			Type:     to.StringPtr("Microsoft.Resources/resourceGroups"),
		}

		clientSet := getFakeClientSetWithAvailabilitySetsClient(*credentials, testLocation, cluster, existingAvailabilitySet, fakeClientModeOkay)
		cluster, err = reconcileAvailabilitySet(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err != nil {
			t.Fatalf("expected reconcileAvailabilitySet to succeed, but failed with error: %v", err)
		}

		if cluster.Spec.Cloud.Azure.AvailabilitySet != customExistingAvailabilitySet {
			t.Fatalf("expected availability set in cloud spec to be '%s', got '%s'", customExistingAvailabilitySet, cluster.Spec.Cloud.Azure.AvailabilitySet)
		}

		fakeClient, ok := clientSet.AvailabilitySets.(*fakeAvailabilitySetsClient)
		if !ok {
			t.Fatalf("failed to access underlying fake AvailabilitySetsClient")
		}

		if fakeClient.CreateOrUpdateCalledCount != 0 {
			t.Fatalf("expected no attempts to update availability set, got %d calls to CreateOrUpdate", fakeClient.CreateOrUpdateCalledCount)
		}
	})

	t.Run("custom-availability-set-does-not-exist", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AzureCloudSpec{
			ResourceGroup:   customExistingResourceGroup,
			AvailabilitySet: "does-not-exist",
		}, credentials)

		clientSet := getFakeClientSetWithAvailabilitySetsClient(*credentials, testLocation, cluster, nil, fakeClientModeOkay)
		cluster, err = reconcileAvailabilitySet(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err != nil {
			t.Fatalf("expected reconcileAvailabilitySet to succeed, but failed with error: %v", err)
		}

		if cluster.Spec.Cloud.Azure.AvailabilitySet != "does-not-exist" {
			t.Fatalf("expected availability set in cloud spec to be '%s', got '%s'", "does-not-exist", cluster.Spec.Cloud.Azure.AvailabilitySet)
		}

		fakeClient, ok := clientSet.AvailabilitySets.(*fakeAvailabilitySetsClient)
		if !ok {
			t.Fatalf("failed to access underlying fake AvailabilitySetsClient")
		}

		if fakeClient.CreateOrUpdateCalledCount != 1 {
			t.Fatalf("expected call to CreateOrUpdate, got %d calls", fakeClient.CreateOrUpdateCalledCount)
		}
	})

	t.Run("invalid-credentials", func(t *testing.T) {
		cluster := makeCluster(&kubermaticv1.AzureCloudSpec{}, credentials)

		clientSet := getFakeClientSetWithAvailabilitySetsClient(*credentials, testLocation, cluster, nil, fakeClientModeAuthFail)
		_, err := reconcileAvailabilitySet(ctx, clientSet, testLocation, cluster, testClusterUpdater(cluster))

		if err == nil {
			t.Fatalf("expected error for request that got a 403 error, got none")
		}
	})
}

const customExistingAvailabilitySet = "custom-existing-availability-set"

type fakeAvailabilitySetsClient struct {
	compute.AvailabilitySetsClient
	location    string
	credentials Credentials
	mode        fakeClientMode
	cluster     *kubermaticv1.Cluster

	AvailabilitySet *compute.AvailabilitySet

	CreateOrUpdateCalledCount int
}

func getFakeClientSetWithAvailabilitySetsClient(credentials Credentials, location string, cluster *kubermaticv1.Cluster, existingAvailabilitySet *compute.AvailabilitySet, mode fakeClientMode) *ClientSet {
	return &ClientSet{
		AvailabilitySets: &fakeAvailabilitySetsClient{
			location:        location,
			credentials:     credentials,
			mode:            mode,
			cluster:         cluster,
			AvailabilitySet: existingAvailabilitySet,
		},
	}
}

func (c *fakeAvailabilitySetsClient) Get(ctx context.Context, resourceGroupName string, availabilitySetName string) (compute.AvailabilitySet, error) {
	switch c.mode {
	case fakeClientModeOkay:
		if c.AvailabilitySet != nil && c.AvailabilitySet.Name != nil && availabilitySetName == *c.AvailabilitySet.Name {
			return *c.AvailabilitySet, nil
		} else {
			resp := autorest.Response{
				Response: &http.Response{
					StatusCode: 404,
				},
			}

			return compute.AvailabilitySet{
				Response: resp,
			}, autorest.NewErrorWithError(fmt.Errorf("not found"), "resources.GroupsClient", "Get", resp.Response, "Failure responding to request")
		}

	case fakeClientModeAuthFail:
		resp := autorest.Response{
			Response: &http.Response{
				StatusCode: 403,
			},
		}

		return compute.AvailabilitySet{
			Response: resp,
		}, autorest.NewErrorWithError(fmt.Errorf("unauthorized"), "resources.GroupsClient", "Get", resp.Response, "Failure responding to request")
	}

	return compute.AvailabilitySet{}, fmt.Errorf("unknown fake client mode: %s", c.mode)

}

func (c *fakeAvailabilitySetsClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, availabilitySetName string, parameters compute.AvailabilitySet) (compute.AvailabilitySet, error) {
	c.CreateOrUpdateCalledCount++
	c.AvailabilitySet = &parameters
	return *c.AvailabilitySet, nil
}
