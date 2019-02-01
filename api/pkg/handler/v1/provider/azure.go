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

var NewAzureClientSet = func(subscriptionID, clientID, clientSecret, tenantID string) (AzureClientSet, error) {
	var err error
	sizesClient := compute.NewVirtualMachineSizesClient(subscriptionID)
	sizesClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, err
	}
	skusClient := compute.NewResourceSkusClient(subscriptionID)
	skusClient.Authorizer, err = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID).Authorizer()
	if err != nil {
		return nil, err
	}

	return &azureClientSetImpl{
		vmSizeClient: sizesClient,
		skusClient:   skusClient,
	}, nil
}

type azureClientSetImpl struct {
	vmSizeClient compute.VirtualMachineSizesClient
	skusClient   compute.ResourceSkusClient
}

type AzureClientSet interface {
	ListVMSize(ctx context.Context, location string) ([]compute.VirtualMachineSize, error)
	ListSKU(ctx context.Context, location string) ([]compute.ResourceSku, error)
}

func (s *azureClientSetImpl) ListSKU(ctx context.Context, location string) ([]compute.ResourceSku, error) {
	skuList, err := s.skusClient.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list SKU resource: %v", err)
	}
	return skuList.Values(), nil
}

func (s *azureClientSetImpl) ListVMSize(ctx context.Context, location string) ([]compute.VirtualMachineSize, error) {
	sizesResult, err := s.vmSizeClient.List(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list sizes: %v", err)
	}
	return *sizesResult.Value, nil
}

func isTierStandard(sku compute.ResourceSku) bool {
	tier := sku.Tier
	if tier != nil {
		if *tier == "Standard" {
			return true
		}
	}
	return false
}

func isVirtualMachinesType(sku compute.ResourceSku) bool {
	resourceType := sku.ResourceType
	if resourceType != nil {
		if *resourceType == "virtualMachines" {
			return true
		}
	}
	return false
}

func isLocation(sku compute.ResourceSku, location string) bool {
	if sku.Locations != nil {
		for _, l := range *sku.Locations {
			if l == location {
				return true
			}
		}
	}
	return false
}

// isValidVM checks all constrains for VM
func isValidVM(sku compute.ResourceSku, location string) bool {

	if !isLocation(sku, location) {
		return false
	}

	if !isTierStandard(sku) {
		return false
	}

	if !isVirtualMachinesType(sku) {
		return false
	}

	// check restricted locations
	restrictions := sku.Restrictions
	if restrictions != nil {
		for _, r := range *restrictions {
			restrictionInfo := r.RestrictionInfo
			if restrictionInfo != nil {
				if restrictionInfo.Locations != nil {
					for _, l := range *restrictionInfo.Locations {
						if l == location {
							return false
						}
					}
				}
			}
		}
	}

	return true
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
	sizesClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for size client: %v", err)
	}

	skuList, err := sizesClient.ListSKU(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list SKU resource: %v", err)
	}

	// prepare set of valid VM size types from SKU resources
	validSKUSet := make(map[string]struct{}, len(skuList))
	for _, v := range skuList {
		if isValidVM(v, location) {
			validSKUSet[*v.Name] = struct{}{}
		}
	}

	// prepare set of valid VM size types for container purpose
	validVMSizeList := compute.PossibleContainerServiceVMSizeTypesValues()
	validVMContainerSet := make(map[string]struct{}, len(validVMSizeList))
	for _, s := range validVMSizeList {
		validVMContainerSet[string(s)] = struct{}{}
	}

	// get all available VM size types for given location
	listVMSize, err := sizesClient.ListVMSize(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list sizes: %v", err)
	}

	var sizeList apiv1.AzureSizeList
	for _, v := range listVMSize {
		if v.Name != nil {
			_, okSKU := validSKUSet[*v.Name]
			_, okVMContainer := validVMContainerSet[*v.Name]

			if okSKU && okVMContainer {
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
