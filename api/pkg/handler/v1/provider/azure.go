package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/dc"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

var NewSizeClient = func(subscriptionID, clientID, clientSecret, tenantID string) (SizeClient, error) {
	var err error
	sizesClient := compute.NewVirtualMachineSizesClient(subscriptionID)
	sizesClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, err
	}
	return &sizeClientImpl{
		vmSizeClient: sizesClient,
	}, nil
}

type sizeClientImpl struct {
	vmSizeClient compute.VirtualMachineSizesClient
}

type SizeClient interface {
	List(ctx context.Context, location string) (compute.VirtualMachineSizeListResult, error)
}

func (s *sizeClientImpl) List(ctx context.Context, location string) (compute.VirtualMachineSizeListResult, error) {
	return s.vmSizeClient.List(ctx, location)
}

func AzureSizeNoCredentialsEndpoint(projectProvider provider.ProjectProvider, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AzureSizeNoCredentialsReq)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
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

func AzureSizeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AzureSizeReq)
		return azureSize(ctx, req.SubscriptionID, req.ClientID, req.ClientSecret, req.TenantID, req.Location)
	}
}

func azureSize(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location string) (apiv1.AzureSizeList, error) {
	sizesClient, err := NewSizeClient(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for size client: %v", err)
	}

	// get all available VM size types for given location
	sizesResult, err := sizesClient.List(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list sizes: %v", err)
	}

	if sizesResult.Value == nil {
		return nil, fmt.Errorf("failed to list sizes: Azure return a nil result")
	}

	// prepare set of valid VM size types
	validVMSizeList := compute.PossibleContainerServiceVMSizeTypesValues()
	validVMSizeSet := make(map[string]struct{}, len(validVMSizeList))
	for _, s := range validVMSizeList {
		validVMSizeSet[string(s)] = struct{}{}
	}

	var sizeList apiv1.AzureSizeList
	for _, v := range *sizesResult.Value {
		// add only valid VM size types
		if v.Name != nil {
			if _, ok := validVMSizeSet[*v.Name]; ok {
				s := apiv1.AzureSize{
					Name:                 *v.Name,
					NumberOfCores:        *v.NumberOfCores,
					OsDiskSizeInMB:       *v.OsDiskSizeInMB,
					ResourceDiskSizeInMB: *v.ResourceDiskSizeInMB,
					MemoryInMB:           *v.MemoryInMB,
					MaxDataDiskCount:     *v.MaxDataDiskCount,
				}

				sizeList = append(sizeList, s)
			}
		}
	}

	return sizeList, nil
}

// AzureSizeNoCredentialsReq represent a request for Azure VM sizes
// note that the request doesn't have credentials for authN
// swagger:parameters listAzureSizesNoCredentials
type AzureSizeNoCredentialsReq struct {
	common.GetClusterReq
}

func DecodeAzureSizesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AzureSizeNoCredentialsReq
	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)
	return req, nil
}

// AzureSizeReq represent a request for Azure VM sizes
type AzureSizeReq struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string
	Location       string
}

func DecodeAzureSizesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AzureSizeReq

	req.SubscriptionID = r.Header.Get("SubscriptionID")
	req.TenantID = r.Header.Get("TenantID")
	req.ClientID = r.Header.Get("ClientID")
	req.ClientSecret = r.Header.Get("ClientSecret")
	req.Location = r.Header.Get("Location")
	return req, nil
}
