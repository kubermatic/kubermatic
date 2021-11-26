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

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func AzureSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(azureSizeNoCredentialsReq)
		return providercommon.AzureSizeWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, settingsProvider, req.ProjectID, req.ClusterID)
	}
}

func AzureAvailabilityZonesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(azureAvailabilityZonesNoCredentialsReq)
		return providercommon.AzureAvailabilityZonesWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID, req.SKUName)
	}
}

func AzureSecurityGroupsEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(azureSecurityGroupsReq)
		credentials, err := getAzureCredentialsFromReq(ctx, req.azureCommonReq, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.AzureSecurityGroupEndpoint(ctx, credentials.subscriptionID, credentials.clientID, credentials.clientSecret, credentials.tenantID, req.Location, req.ResourceGroup)
	}
}

func AzureResourceGroupsEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(azureResourceGroupsReq)
		credentials, err := getAzureCredentialsFromReq(ctx, req.azureCommonReq, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.AzureResourceGroupEndpoint(ctx, credentials.subscriptionID, credentials.clientID, credentials.clientSecret, credentials.tenantID, req.Location)
	}
}

func AzureRouteTablesEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(azureRouteTablesReq)
		credentials, err := getAzureCredentialsFromReq(ctx, req.azureCommonReq, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.AzureRouteTableEndpoint(ctx, credentials.subscriptionID, credentials.clientID, credentials.clientSecret, credentials.tenantID, req.Location, req.ResourceGroup)
	}
}

func AzureVirtualNetworksEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(azureVirtualNetworksReq)
		credentials, err := getAzureCredentialsFromReq(ctx, req.azureCommonReq, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.AzureVnetEndpoint(ctx, credentials.subscriptionID, credentials.clientID, credentials.clientSecret, credentials.tenantID, req.Location, req.ResourceGroup)
	}
}

func AzureSubnetsEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(azureSubnetsReq)
		credentials, err := getAzureCredentialsFromReq(ctx, req.azureCommonReq, userInfoGetter, presetsProvider)
		if err != nil {
			return nil, err
		}
		return providercommon.AzureSubnetEndpoint(ctx, credentials.subscriptionID, credentials.clientID, credentials.clientSecret, credentials.tenantID, req.ResourceGroup, req.VirtualNetwork)
	}
}

type azureCredentials struct {
	subscriptionID string
	tenantID       string
	clientID       string
	clientSecret   string
}

func getAzureCredentialsFromReq(ctx context.Context, req azureCommonReq, userInfoGetter provider.UserInfoGetter, presetsProvider provider.PresetProvider) (*azureCredentials, error) {
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

	return &azureCredentials{
		subscriptionID: subscriptionID,
		tenantID:       tenantID,
		clientID:       clientID,
		clientSecret:   clientSecret,
	}, nil
}

// azureSizeNoCredentialsReq represent a request for Azure VM sizes
// note that the request doesn't have credentials for authN
// swagger:parameters listAzureSizesNoCredentialsV2
type azureSizeNoCredentialsReq struct {
	cluster.GetClusterReq
}

// GetSeedCluster returns the SeedCluster object
func (req azureSizeNoCredentialsReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeAzureSizesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req azureSizeNoCredentialsReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)
	return req, nil
}

// azureAvailabilityZonesNoCredentialsReq represent a request for Azure Availability Zones
// note that the request doesn't have credentials for authN
// swagger:parameters listAzureAvailabilityZonesNoCredentialsV2
type azureAvailabilityZonesNoCredentialsReq struct {
	azureSizeNoCredentialsReq
	// in: header
	// name: SKUName
	SKUName string
}

// GetSeedCluster returns the SeedCluster object
func (req azureAvailabilityZonesNoCredentialsReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeAzureAvailabilityZonesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req azureAvailabilityZonesNoCredentialsReq
	lr, err := DecodeAzureSizesNoCredentialsReq(c, r)
	if err != nil {
		return nil, err
	}
	req.azureSizeNoCredentialsReq = lr.(azureSizeNoCredentialsReq)
	req.SKUName = r.Header.Get("SKUName")
	return req, nil
}

// azureCommonReq represent a request for Azure support
type azureCommonReq struct {
	// in: header
	SubscriptionID string
	// in: header
	TenantID string
	// in: header
	ClientID string
	// in: header
	ClientSecret string
	// in: header
	// Credential predefined Kubermatic credential name from the presets
	Credential string
}

func DecodeAzureCommonReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req azureCommonReq

	req.SubscriptionID = r.Header.Get("SubscriptionID")
	req.TenantID = r.Header.Get("TenantID")
	req.ClientID = r.Header.Get("ClientID")
	req.ClientSecret = r.Header.Get("ClientSecret")
	req.Credential = r.Header.Get("Credential")
	return req, nil
}

// azureSecurityGroupsReq represent a request for Azure VM security groups
// swagger:parameters listAzureSecurityGroups
type azureSecurityGroupsReq struct {
	azureCommonReq

	// in: header
	ResourceGroup string
	// in: header
	Location string
}

func DecodeAzureSecurityGroupsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req azureSecurityGroupsReq
	common, err := DecodeAzureCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.azureCommonReq = common.(azureCommonReq)
	req.ResourceGroup = r.Header.Get("ResourceGroup")
	req.Location = r.Header.Get("Location")
	return req, nil
}

// azureResourceGroupsReq represent a request for Azure VM resource groups
// swagger:parameters listAzureResourceGroups
type azureResourceGroupsReq struct {
	azureCommonReq

	// in: header
	Location string
}

func DecodeAzureResourceGroupsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req azureResourceGroupsReq
	common, err := DecodeAzureCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.azureCommonReq = common.(azureCommonReq)

	req.Location = r.Header.Get("Location")
	return req, nil
}

// azureRouteTablesReq represent a request for Azure VM route tables
// swagger:parameters listAzureRouteTables
type azureRouteTablesReq struct {
	azureCommonReq

	// in: header
	ResourceGroup string
	// in: header
	Location string
}

func DecodeAzureRouteTablesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req azureRouteTablesReq
	common, err := DecodeAzureCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.azureCommonReq = common.(azureCommonReq)
	req.ResourceGroup = r.Header.Get("ResourceGroup")
	req.Location = r.Header.Get("Location")
	return req, nil
}

// azureVirtualNetworksReq represent a request for Azure VM virtual networks
// swagger:parameters listAzureVnets
type azureVirtualNetworksReq struct {
	azureCommonReq

	// in: header
	ResourceGroup string
	// in: header
	Location string
}

func DecodeAzureVirtualNetworksReq(c context.Context, r *http.Request) (interface{}, error) {
	var req azureVirtualNetworksReq
	common, err := DecodeAzureCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.azureCommonReq = common.(azureCommonReq)
	req.ResourceGroup = r.Header.Get("ResourceGroup")
	req.Location = r.Header.Get("Location")
	return req, nil
}

// azureSubnetsReq represent a request for Azure VM subnets
// swagger:parameters listAzureSubnets
type azureSubnetsReq struct {
	azureCommonReq

	// in: header
	ResourceGroup string
	// in: header
	VirtualNetwork string
}

func DecodeAzureSubnetsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req azureSubnetsReq
	common, err := DecodeAzureCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.azureCommonReq = common.(azureCommonReq)
	req.ResourceGroup = r.Header.Get("ResourceGroup")
	req.VirtualNetwork = r.Header.Get("VirtualNetwork")
	return req, nil
}

// AKSTypesReq represent a request for AKS types.
// swagger:parameters listAKSClusters
type AKSTypesReq struct {
	AKSCommonReq
}

// AKSCommonReq represent a request with common parameters for AKS.
type AKSCommonReq struct {
	// in: header
	// name: TenantID
	TenantID string
	// in: header
	// name: SubscriptionID
	SubscriptionID string
	// in: header
	// name: ClientID
	ClientID string
	// in: header
	// name: ClientSecret
	ClientSecret string
	// in: header
	// name: Credential
	Credential string
}

// Validate validates AKSCommonReq request
func (req AKSCommonReq) Validate() error {
	if len(req.Credential) == 0 && len(req.TenantID) == 0 && len(req.SubscriptionID) == 0 && len(req.ClientID) == 0 && len(req.ClientSecret) == 0 {
		return fmt.Errorf("Azure credentials cannot be empty")
	}
	return nil
}

func DecodeAKSCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AKSCommonReq

	req.TenantID = r.Header.Get("TenantID")
	req.SubscriptionID = r.Header.Get("SubscriptionID")
	req.ClientID = r.Header.Get("ClientID")
	req.ClientSecret = r.Header.Get("ClientSecret")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func DecodeAKSTypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AKSTypesReq

	commonReq, err := DecodeAKSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.AKSCommonReq = commonReq.(AKSCommonReq)

	return req, nil
}

func ListAKSClustersEndpoint(userInfoGetter provider.UserInfoGetter, presetsProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(AKSTypesReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		credential := providercommon.AzureCredential{
			TenantID:       req.TenantID,
			SubscriptionID: req.SubscriptionID,
			ClientID:       req.ClientID,
			ClientSecret:   req.ClientSecret,
		}
		presetName := req.Credential

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Preset is used
		if len(presetName) > 0 {
			credential, err = getAzurePresetCredentials(userInfo, presetName, presetsProvider)
			if err != nil {
				return nil, fmt.Errorf("error getting preset credentials for Azure: %v", err)
			}
		}
		return providercommon.ListAKSClusters(ctx, credential)
	}
}

func getAzurePresetCredentials(userInfo *provider.UserInfo, presetName string, presetProvider provider.PresetProvider) (providercommon.AzureCredential, error) {

	preset, err := presetProvider.GetPreset(userInfo, presetName)
	if err != nil {
		return providercommon.AzureCredential{}, fmt.Errorf("can not get preset %s for the user %s", presetName, userInfo.Email)
	}

	azure := preset.Spec.Azure
	if azure == nil {
		return providercommon.AzureCredential{}, fmt.Errorf("credentials for Azure not present in preset %s for the user %s", presetName, userInfo.Email)
	}
	return providercommon.AzureCredential{
		TenantID:       azure.TenantID,
		SubscriptionID: azure.SubscriptionID,
		ClientID:       azure.ClientID,
		ClientSecret:   azure.ClientSecret,
	}, nil
}
