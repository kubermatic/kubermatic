package provider_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	azure "github.com/kubermatic/kubermatic/api/pkg/handler/v1/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

const (
	testID     = "test"
	locationUS = "US"
	locationEU = "EU"
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
				{"name":"Standard_GS3", "maxDataDiskCount": 3, "memoryInMB": 254, "numberOfCores": 8, "osDiskSizeInMB": 1024, "resourceDiskSizeInMB":1024},
				{"name":"Standard_A5", "maxDataDiskCount": 3, "memoryInMB": 254, "numberOfCores": 8, "osDiskSizeInMB": 1024, "resourceDiskSizeInMB":1024}
			]`,
		},
		{
			name:       "test EU location when only one VM size type is valid",
			httpStatus: http.StatusOK,
			location:   locationEU,
			secret:     "secret",
			expectedResponse: `[
				{"name":"Standard_GS3", "maxDataDiskCount": 3, "memoryInMB": 254, "numberOfCores": 8, "osDiskSizeInMB": 1024, "resourceDiskSizeInMB":1024}
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

			azure.NewSizeClient = MockNewSizeClient

			apiUser := test.GetUser(test.UserEmail, test.UserID, test.UserName, false)

			res := httptest.NewRecorder()
			router, _, err := test.CreateTestEndpointAndGetClients(apiUser, buildAzureDatacenterMeta(), []runtime.Object{}, []runtime.Object{}, []runtime.Object{test.APIUserToKubermaticUser(apiUser)}, nil, nil, hack.NewTestRouting)
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

func buildAzureDatacenterMeta() map[string]provider.DatacenterMeta {
	return map[string]provider.DatacenterMeta{
		datacenterName: {
			Location: "ap-northeast",
			Country:  "JP",
			Private:  false,
			IsSeed:   true,
			Spec: provider.DatacenterSpec{
				Azure: &provider.AzureSpec{
					Location: "ap-northeast",
				},
			},
		},
	}
}

func MockNewSizeClient(subscriptionID, clientID, clientSecret, tenantID string) (azure.SizeClient, error) {

	if len(clientSecret) == 0 || len(subscriptionID) == 0 || len(clientID) == 0 || len(tenantID) == 0 {
		return nil, fmt.Errorf("")
	}

	return &mockSizeClientImpl{}, nil
}

func (s *mockSizeClientImpl) List(ctx context.Context, location string) (compute.VirtualMachineSizeListResult, error) {

	standardFake := "Fake"
	standardGS3 := "Standard_GS3"
	standardA5 := "Standard_A5"
	maxDataDiskCount := int32(3)
	memoryInMB := int32(254)
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

	return s.machineSizeList, nil
}
