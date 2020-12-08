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

	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

func AzureSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AzureSizeNoCredentialsReq)
		return providercommon.AzureSizeWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
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
		return providercommon.AzureSize(ctx, subscriptionID, clientID, clientSecret, tenantID, req.Location)
	}
}

func AzureAvailabilityZonesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AzureAvailabilityZonesNoCredentialsReq)
		return providercommon.AzureAvailabilityZonesWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID, req.SKUName)
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
		return providercommon.AzureSKUAvailabilityZones(ctx, subscriptionID, clientID, clientSecret, tenantID, location, skuName)
	}
}

// AvailabilityZonesReq represent a request for Azure VM Multi-AvailabilityZones support
// swagger:parameters listAzureSKUAvailabilityZones
type AvailabilityZonesReq struct {
	// in: header
	SubscriptionID string
	// in: header
	TenantID string
	// in: header
	ClientID string
	// in: header
	ClientSecret string
	// in: header
	Location string
	// in: header
	SKUName string
	// in: header
	// Credential predefined Kubermatic credential name from the presets
	Credential string
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
// swagger:parameters listAzureSizes
type AzureSizeReq struct {
	// in: header
	SubscriptionID string
	// in: header
	TenantID string
	// in: header
	ClientID string
	// in: header
	ClientSecret string
	// in: header
	Location string
	// in: header
	// Credential predefined Kubermatic credential name from the presets
	Credential string
}

func DecodeAzureSizesReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req AzureSizeReq

	req.SubscriptionID = r.Header.Get("SubscriptionID")
	req.TenantID = r.Header.Get("TenantID")
	req.ClientID = r.Header.Get("ClientID")
	req.ClientSecret = r.Header.Get("ClientSecret")
	req.Location = r.Header.Get("Location")
	req.Credential = r.Header.Get("Credential")
	return req, nil
}

func DecodeAzureAvailabilityZonesReq(_ context.Context, r *http.Request) (interface{}, error) {
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
