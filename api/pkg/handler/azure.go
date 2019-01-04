package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/dc"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func azureSizeNoCredentialsEndpoint(projectProvider provider.ProjectProvider, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AzureSizeNoCredentialsReq)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		if cluster.Spec.Cloud.Azure == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		dc, err := dc.GetDatacenter(dcs, cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		if dc.Spec.Azure == nil {
			return nil, errors.NewNotFound("cloud spec (dc) for ", req.ClusterID)
		}

		azureSpec := cluster.Spec.Cloud.Azure
		azureLocation := dc.Spec.Azure.Location
		return azureSize(ctx, azureSpec.SubscriptionID, azureSpec.ClientID, azureSpec.ClientSecret, azureSpec.TenantID, azureLocation)
	}
}

func azureSizeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AzureSizeReq)
		return azureSize(ctx, req.SubscriptionID, req.ClientID, req.ClientSecret, req.TenantID, req.Location)
	}
}

func azureSize(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location string) (apiv1.AzureSizeList, error) {
	var err error
	sizesClient := compute.NewVirtualMachineSizesClient(subscriptionID)
	sizesClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %v", err)
	}

	sizesResult, err := sizesClient.List(ctx, location)
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
