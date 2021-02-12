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

func HetznerSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(HetznerSizesNoCredentialsReq)
		return providercommon.HetznerSizeWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, settingsProvider, req.ProjectID, req.ClusterID)
	}
}

func HetznerSizeEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(HetznerSizesReq)
		token := req.HetznerToken

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Hetzner; credentials != nil {
				token = credentials.Token
			}
		}
		settings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return providercommon.HetznerSize(ctx, settings.Spec.MachineDeploymentVMResourceQuota, token)
	}
}

// HetznerSizesNoCredentialsReq represent a request for hetzner sizes EP
// swagger:parameters listHetznerSizesNoCredentials
type HetznerSizesNoCredentialsReq struct {
	common.GetClusterReq
}

func DecodeHetznerSizesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req HetznerSizesNoCredentialsReq
	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)
	return req, nil
}

// HetznerSizesReq represent a request for hetzner sizes
// swagger:parameters listHetznerSizes
type HetznerSizesReq struct {
	// in: header
	// HetznerToken Hetzner token
	HetznerToken string
	// in: header
	// Credential predefined Kubermatic credential name from the presets
	Credential string
}

func DecodeHetznerSizesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req HetznerSizesReq

	req.HetznerToken = r.Header.Get("HetznerToken")
	req.Credential = r.Header.Get("Credential")
	return req, nil
}
