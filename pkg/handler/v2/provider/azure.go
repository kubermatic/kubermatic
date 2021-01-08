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
)

func AzureSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(azureSizeNoCredentialsReq)
		return providercommon.AzureSizeWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
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
