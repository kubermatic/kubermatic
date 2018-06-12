package handler

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
)

func azureSizeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AzureSizeReq)

		var err error
		sizesClient := compute.NewVirtualMachineSizesClient(req.SubscriptionID)
		sizesClient.Authorizer, err = auth.NewClientCredentialsConfig(req.ClientID, req.ClientSecret, req.TenantID).Authorizer()
		if err != nil {
			return nil, fmt.Errorf("failed to create authorizer: %v", err)
		}

		sizesResult, err := sizesClient.List(ctx, req.Location)
		if err != nil {
			return nil, fmt.Errorf("failed to list sizes: %v", err)
		}

		if sizesResult.Value == nil {
			return nil, fmt.Errorf("failed to list sizes: Azure return a nil result")
		}

		var sizeList apiv1.AzureSizeList
		for _, v := range *sizesResult.Value {
			s := apiv1.AzureSize{
				Name:                 v.Name,
				NumberOfCores:        v.NumberOfCores,
				OsDiskSizeInMB:       v.OsDiskSizeInMB,
				ResourceDiskSizeInMB: v.ResourceDiskSizeInMB,
				MemoryInMB:           v.MemoryInMB,
				MaxDataDiskCount:     v.MaxDataDiskCount,
			}

			sizeList = append(sizeList, s)
		}

		return sizeList, nil
	}
}
