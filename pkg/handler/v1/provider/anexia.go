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
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

func AnexiaVlanEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AnexiaReq)

		token := req.Token
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Anexia; credentials != nil {
				token = credentials.Token
			}
		}

		return providercommon.ListAnexiaVlans(ctx, token)
	}
}

func AnexiaTemplateEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AnexiaTemplateReq)

		token := req.Token
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Anexia; credentials != nil {
				token = credentials.Token
			}
		}

		return providercommon.ListAnexiaTemplates(ctx, token, req.Location)
	}
}

// AnexiaTemplateReq represent a request for Anexia template resources
// swagger:parameters listAnexiaTemplates
type AnexiaTemplateReq struct {
	AnexiaReq

	// in: header
	// Location Anexia location ID
	Location string
}

// AnexiaReq represent a request for Anexia resources
// swagger:parameters listAnexiaVlans
type AnexiaReq struct {
	// in: header
	// Token Anexia token
	Token string
	// in: header
	// Credential predefined Kubermatic credential name from the presets
	Credential string
}

func DecodeAnexiaReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req AnexiaReq

	req.Token = r.Header.Get("Token")
	req.Credential = r.Header.Get("Credential")
	return req, nil
}

func DecodeAnexiaTemplateReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req AnexiaTemplateReq

	req.Token = r.Header.Get("Token")
	req.Credential = r.Header.Get("Credential")
	req.Location = r.Header.Get("Location")
	return req, nil
}
