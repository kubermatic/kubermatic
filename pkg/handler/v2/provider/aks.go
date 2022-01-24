/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// AKSTypesReq represent a request for AKS types.
// swagger:parameters validateAKSCredentials
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

// Validate validates aksCommonReq request
func (req AKSCommonReq) Validate() error {
	if len(req.Credential) == 0 && len(req.TenantID) == 0 && len(req.SubscriptionID) == 0 && len(req.ClientID) == 0 && len(req.ClientSecret) == 0 {
		return fmt.Errorf("AKS credentials cannot be empty")
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

func DecodeAKSClusterListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AKSClusterListReq

	commonReq, err := DecodeAKSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.AKSCommonReq = commonReq.(AKSCommonReq)
	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	return req, nil
}

// AKSClusterListReq represent a request for AKS cluster list.
// swagger:parameters listAKSClusters
type AKSClusterListReq struct {
	common.ProjectReq
	AKSCommonReq
}

func ListAKSClustersEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AKSClusterListReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		cred, err := getAKSCredentialsFromReq(ctx, req.AKSCommonReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return providercommon.ListAKSClusters(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, clusterProvider, *cred, req.ProjectID)
	}
}

func AKSValidateCredentialsEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AKSTypesReq)

		cred, err := getAKSCredentialsFromReq(ctx, req.AKSCommonReq, userInfoGetter, presetProvider)
		if err != nil {
			return nil, err
		}

		return nil, providercommon.ValidateAKSCredentials(ctx, *cred)
	}
}

func getAKSCredentialsFromReq(ctx context.Context, req AKSCommonReq, userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider) (*resources.AKSCredentials, error) {
	subscriptionID := req.SubscriptionID
	clientID := req.ClientID
	clientSecret := req.ClientSecret
	tenantID := req.TenantID

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	if len(req.Credential) > 0 {
		preset, err := presetProvider.GetPreset(userInfo, req.Credential)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
		}
		if credentials := preset.Spec.AKS; credentials != nil {
			subscriptionID = credentials.SubscriptionID
			clientID = credentials.ClientID
			clientSecret = credentials.ClientSecret
			tenantID = credentials.TenantID
		}
	}

	return &resources.AKSCredentials{
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
	}, nil
}
