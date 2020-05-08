package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/dc"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/azure"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
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

func AzureSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AzureSizeNoCredentialsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.Azure == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		dc, err := dc.GetDatacenter(userInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		if dc.Spec.Azure == nil {
			return nil, errors.NewNotFound("cloud spec (dc) for ", req.ClusterID)
		}

		azureLocation := dc.Spec.Azure.Location
		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
		creds, err := azure.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
		if err != nil {
			return nil, err
		}
		return azureSize(ctx, creds.SubscriptionID, creds.ClientID, creds.ClientSecret, creds.TenantID, azureLocation)
	}
}

func AzureSizeEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AzureSizeReq)

		subscriptionID := req.SubscriptionID
		clientID := req.ClientID
		clientSecret := req.ClientSecret
		tenantID := req.TenantID

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Azure; credentials != nil {
				subscriptionID = credentials.SubscriptionID
				clientID = credentials.ClientID
				clientSecret = credentials.ClientSecret
				tenantID = credentials.TenantID
			}
		}
		return azureSize(ctx, subscriptionID, clientID, clientSecret, tenantID, req.Location)
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
					Name:          *v.Name,
					NumberOfCores: *v.NumberOfCores,
					// TODO: Use this to validate user-defined disk size.
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

func AzureAvailabilityZonesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AzureAvailabilityZonesNoCredentialsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.Azure == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		dc, err := dc.GetDatacenter(userInfo, seedsGetter, cluster.Spec.Cloud.DatacenterName)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		if dc.Spec.Azure == nil {
			return nil, errors.NewNotFound("cloud spec (dc) for ", req.ClusterID)
		}

		azureLocation := dc.Spec.Azure.Location
		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
		creds, err := azure.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
		if err != nil {
			return nil, err
		}
		return azureSKUAvailabilityZones(ctx, creds.SubscriptionID, creds.ClientID, creds.ClientSecret, creds.TenantID, azureLocation, req.SKUName)
	}
}

func AzureAvailabilityZonesEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AvailabilityZonesReq)

		subscriptionID := req.SubscriptionID
		clientID := req.ClientID
		clientSecret := req.ClientSecret
		tenantID := req.TenantID
		location := req.Location
		skuName := req.SKUName

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Azure; credentials != nil {
				subscriptionID = credentials.SubscriptionID
				clientID = credentials.ClientID
				clientSecret = credentials.ClientSecret
				tenantID = credentials.TenantID
			}
		}
		return azureSKUAvailabilityZones(ctx, subscriptionID, clientID, clientSecret, tenantID, location, skuName)
	}
}

// AvailabilityZonesReq represent a request for Azure VM Multi-AvailabilityZones support
type AvailabilityZonesReq struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string
	Location       string
	SKUName        string
	Credential     string
}

func azureSKUAvailabilityZones(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location, skuName string) (*apiv1.AzureAvailabilityZonesList, error) {
	azSKUClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for sku client: %v", err)
	}

	skuList, err := azSKUClient.ListSKU(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list sku resource: %v", err)
	}

	var azZones = &apiv1.AzureAvailabilityZonesList{}
	for _, sku := range skuList {
		if skuName == *sku.Name {
			for _, l := range *sku.LocationInfo {
				if location == *l.Location {
					if *l.Zones != nil && len(*l.Zones) > 0 {
						azZones.Zones = *l.Zones
						return azZones, nil
					}
				}
			}
		}
	}

	return nil, nil
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
	Credential     string
}

func DecodeAzureSizesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AzureSizeReq

	req.SubscriptionID = r.Header.Get("SubscriptionID")
	req.TenantID = r.Header.Get("TenantID")
	req.ClientID = r.Header.Get("ClientID")
	req.ClientSecret = r.Header.Get("ClientSecret")
	req.Location = r.Header.Get("Location")
	req.Credential = r.Header.Get("Credential")
	return req, nil
}

func DecodeAzureAvailabilityZonesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AvailabilityZonesReq

	req.SubscriptionID = r.Header.Get("SubscriptionID")
	req.TenantID = r.Header.Get("TenantID")
	req.ClientID = r.Header.Get("ClientID")
	req.ClientSecret = r.Header.Get("ClientSecret")
	req.Location = r.Header.Get("Location")
	req.SKUName = r.Header.Get("SKUName")
	req.Credential = r.Header.Get("Credential")
	return req, nil
}

// AzureAvailabilityZonesNoCredentialsReq represent a request for Azure Availability Zones
// note that the request doesn't have credentials for authN
// swagger:parameters listAzureAvailabilityZonesNoCredentials
type AzureAvailabilityZonesNoCredentialsReq struct {
	common.GetClusterReq
	// in: header
	// name: SKUName
	SKUName string
}

func DecodeAzureAvailabilityZonesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AzureAvailabilityZonesNoCredentialsReq
	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)
	req.SKUName = r.Header.Get("SKUName")
	return req, nil
}
