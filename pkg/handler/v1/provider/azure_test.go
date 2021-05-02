/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package provider_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-12-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testID      = "test"
	locationUS  = "US"
	locationEU  = "EU"
	standardGS3 = "Standard_GS3"
	standardA5  = "Standard_A5"
)

type mockSizeClientImpl struct {
	machineSizeList compute.VirtualMachineSizeListResult
}

func TestAzureSizeEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name             string
		secret           string
		location         string
		httpStatus       int
		expectedResponse string
	}{
		{
			name:             "test when user unauthorized",
			httpStatus:       http.StatusInternalServerError,
			expectedResponse: "",
		},
		{
			name:       "test US location when two VM size types are valid",
			httpStatus: http.StatusOK,
			location:   locationUS,
			secret:     "secret",
			expectedResponse: `[
				{"name":"Standard_GS3", "maxDataDiskCount": 3, "memoryInMB": 2048, "numberOfCores": 8, "numberOfGPUs": 0, "osDiskSizeInMB": 1024, "resourceDiskSizeInMB":1024},
				{"name":"Standard_A5", "maxDataDiskCount": 3, "memoryInMB": 2048, "numberOfCores": 8, "numberOfGPUs": 0, "osDiskSizeInMB": 1024, "resourceDiskSizeInMB":1024}
			]`,
		},
		{
			name:       "test EU location when only one VM size type is valid",
			httpStatus: http.StatusOK,
			location:   locationEU,
			secret:     "secret",
			expectedResponse: `[
				{"name":"Standard_GS3", "maxDataDiskCount": 3, "memoryInMB": 2048, "numberOfCores": 8, "numberOfGPUs": 0, "osDiskSizeInMB": 1024, "resourceDiskSizeInMB":1024}
			]`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			req := httptest.NewRequest("GET", "/api/v1/providers/azure/sizes", strings.NewReader(""))

			req.Header.Add("SubscriptionID", testID)
			req.Header.Add("ClientID", testID)
			req.Header.Add("ClientSecret", tc.secret)
			req.Header.Add("TenantID", testID)
			req.Header.Add("Location", tc.location)

			providercommon.NewAzureClientSet = MockNewSizeClient

			apiUser := test.GetUser(test.UserEmail, test.UserID, test.UserName)

			res := httptest.NewRecorder()
			router, _, err := test.CreateTestEndpointAndGetClients(apiUser, buildAzureDatacenterMeta(), []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{test.APIUserToKubermaticUser(apiUser)}, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v\n", err)
			}

			router.ServeHTTP(res, req)

			// validate
			assert.Equal(t, tc.httpStatus, res.Code)

			if res.Code == http.StatusOK {
				compareJSON(t, res, tc.expectedResponse)
			}

		})
	}
}

func buildAzureDatacenterMeta() provider.SeedsGetter {
	return func() (map[string]*kubermaticv1.Seed, error) {
		return map[string]*kubermaticv1.Seed{
			"my-seed": {
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						datacenterName: {
							Location: "ap-northeast",
							Country:  "JP",
							Spec: kubermaticv1.DatacenterSpec{
								Azure: &kubermaticv1.DatacenterSpecAzure{
									Location: "ap-northeast",
								},
							},
						},
					},
				},
			},
		}, nil
	}
}

func MockNewSizeClient(subscriptionID, clientID, clientSecret, tenantID string) (providercommon.AzureClientSet, error) {

	if len(clientSecret) == 0 || len(subscriptionID) == 0 || len(clientID) == 0 || len(tenantID) == 0 {
		return nil, fmt.Errorf("")
	}

	return &mockSizeClientImpl{}, nil
}

func (s *mockSizeClientImpl) ListSKU(ctx context.Context, location string) ([]compute.ResourceSku, error) {

	standardGS3 := standardGS3
	standardA5 := standardA5
	resourceType := "virtualMachines"
	tier := "Standard"

	resultList := []compute.ResourceSku{
		{
			Locations:    &[]string{locationEU},
			Name:         &standardGS3,
			ResourceType: &resourceType,
			Tier:         &tier,
		},
		{
			Locations:    &[]string{locationUS},
			Name:         &standardGS3,
			ResourceType: &resourceType,
			Tier:         &tier,
		},
		{
			Locations:    &[]string{locationUS},
			Name:         &standardA5,
			ResourceType: &resourceType,
			Tier:         &tier,
		},
	}

	return resultList, nil
}

func (s *mockSizeClientImpl) ListVMSize(ctx context.Context, location string) ([]compute.VirtualMachineSize, error) {

	standardFake := "Fake"
	standardGS3 := "Standard_GS3"
	standardA5 := "Standard_A5"
	maxDataDiskCount := int32(3)
	memoryInMB := int32(2048)
	numberOfCores := int32(8)
	diskSizeInMB := int32(1024)

	s.machineSizeList = compute.VirtualMachineSizeListResult{Value: &[]compute.VirtualMachineSize{}}

	if location == locationEU {
		// one valid VM size type, two in total
		s.machineSizeList.Value = &[]compute.VirtualMachineSize{{Name: &standardGS3,
			MaxDataDiskCount: &maxDataDiskCount, MemoryInMB: &memoryInMB, NumberOfCores: &numberOfCores,
			OsDiskSizeInMB: &diskSizeInMB, ResourceDiskSizeInMB: &diskSizeInMB},
			{Name: &standardFake,
				MaxDataDiskCount: &maxDataDiskCount, MemoryInMB: &memoryInMB, NumberOfCores: &numberOfCores,
				OsDiskSizeInMB: &diskSizeInMB, ResourceDiskSizeInMB: &diskSizeInMB},
		}
	}
	if location == locationUS {
		// two valid VM size types, three in total
		s.machineSizeList.Value = &[]compute.VirtualMachineSize{
			{Name: &standardGS3, MaxDataDiskCount: &maxDataDiskCount, MemoryInMB: &memoryInMB, NumberOfCores: &numberOfCores,
				OsDiskSizeInMB: &diskSizeInMB, ResourceDiskSizeInMB: &diskSizeInMB},
			{Name: &standardFake,
				MaxDataDiskCount: &maxDataDiskCount, MemoryInMB: &memoryInMB, NumberOfCores: &numberOfCores,
				OsDiskSizeInMB: &diskSizeInMB, ResourceDiskSizeInMB: &diskSizeInMB},
			{Name: &standardA5,
				MaxDataDiskCount: &maxDataDiskCount, MemoryInMB: &memoryInMB, NumberOfCores: &numberOfCores,
				OsDiskSizeInMB: &diskSizeInMB, ResourceDiskSizeInMB: &diskSizeInMB},
		}
	}

	return *s.machineSizeList.Value, nil
}

func (s *mockSizeClientImpl) ListSecurityGroups(_ context.Context, _ string) ([]network.SecurityGroup, error) {
	return nil, nil
}

func (s *mockSizeClientImpl) ListResourceGroups(_ context.Context) ([]resources.Group, error) {
	return nil, nil
}

func (s *mockSizeClientImpl) ListRouteTables(_ context.Context, _ string) ([]network.RouteTable, error) {
	return nil, nil
}

func (s *mockSizeClientImpl) ListVnets(_ context.Context, _ string) ([]network.VirtualNetwork, error) {
	return nil, nil
}

func (s *mockSizeClientImpl) ListSubnets(_ context.Context, _, _ string) ([]network.Subnet, error) {
	return nil, nil
}
