package provider_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	azure "github.com/kubermatic/kubermatic/api/pkg/handler/v1/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

			azure.NewAzureClientSet = MockNewSizeClient

			apiUser := test.GetUser(test.UserEmail, test.UserID, test.UserName)

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

func MockNewSizeClient(subscriptionID, clientID, clientSecret, tenantID string) (azure.AzureClientSet, error) {

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

	return *s.machineSizeList.Value, nil
}
