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

// PacketSizesReq represent a request for Packet sizes.
// swagger:parameters listPacketSizes
type PacketSizesReq struct {
	// in: header
	// name: APIKey
	APIKey string `json:"apiKey"`
	// in: header
	// name: ProjectID
	ProjectID string `json:"projectID"`
	// in: header
	// name: Credential
	Credential string `json:"credential"`
}

// PacketSizesNoCredentialsReq represent a request for Packet sizes EP
// swagger:parameters listPacketSizesNoCredentials
type PacketSizesNoCredentialsReq struct {
	common.GetClusterReq
}

func DecodePacketSizesReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req PacketSizesReq

	req.APIKey = r.Header.Get("apiKey")
	req.ProjectID = r.Header.Get("projectID")
	req.Credential = r.Header.Get("credential")

	return req, nil
}

func DecodePacketSizesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req PacketSizesNoCredentialsReq

	commonReq, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = commonReq.(common.GetClusterReq)

	return req, nil
}

func PacketSizesEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PacketSizesReq)

		projectID := req.ProjectID
		apiKey := req.APIKey

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Packet; credentials != nil {
				projectID = credentials.ProjectID
				apiKey = credentials.APIKey
			}
		}

		settings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return providercommon.PacketSizes(apiKey, projectID, settings.Spec.MachineDeploymentVMResourceQuota)
	}
}

func PacketSizesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PacketSizesNoCredentialsReq)
		return providercommon.PacketSizesWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, settingsProvider, req.ProjectID, req.ClusterID)
	}
}
