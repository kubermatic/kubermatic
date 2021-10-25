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

package addon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"k8s.io/apimachinery/pkg/util/sets"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

// addonReq defines HTTP request for getAddon and deleteAddon
// swagger:parameters getAddon deleteAddon
type addonReq struct {
	common.GetClusterReq
	// in: path
	AddonID string `json:"addon_id"`
}

// listReq defines HTTP request for listAddons and listInstallableAddons endpoints
// swagger:parameters listAddons listInstallableAddons
type listReq struct {
	common.GetClusterReq
}

// createReq defines HTTP request for createAddon endpoint
// swagger:parameters createAddon
type createReq struct {
	common.GetClusterReq
	// in: body
	Body apiv1.Addon
}

// patchReq defines HTTP request for patchAddon endpoint
// swagger:parameters patchAddon
type patchReq struct {
	addonReq
	// in: body
	Body apiv1.Addon
}

// patchReq defines HTTP request for getAddonConfig endpoint
// swagger:parameters getAddonConfig
type getConfigReq struct {
	// in: path
	AddonID string `json:"addon_id"`
}

func DecodeGetAddon(c context.Context, r *http.Request) (interface{}, error) {
	var req addonReq

	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	addonID, err := decodeAddonID(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)
	req.AddonID = addonID

	return req, nil
}

func DecodeListAddons(c context.Context, r *http.Request) (interface{}, error) {
	var req listReq

	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)

	return req, nil
}

func DecodeCreateAddon(c context.Context, r *http.Request) (interface{}, error) {
	var req createReq

	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func DecodePatchAddon(c context.Context, r *http.Request) (interface{}, error) {
	var req patchReq

	gr, err := DecodeGetAddon(c, r)
	if err != nil {
		return nil, err
	}

	req.addonReq = gr.(addonReq)
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func DecodeGetConfig(c context.Context, r *http.Request) (interface{}, error) {
	var req getConfigReq

	addonID, err := decodeAddonID(c, r)
	if err != nil {
		return nil, err
	}

	req.AddonID = addonID

	return req, nil
}

func decodeAddonID(c context.Context, r *http.Request) (string, error) {
	addonID := mux.Vars(r)["addon_id"]
	if addonID == "" {
		return "", fmt.Errorf("'addon_id' parameter is required but was not provided")
	}

	return addonID, nil
}

func ListAccessibleAddons(configGetter provider.KubermaticConfigurationGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		config, err := configGetter(ctx)
		if err != nil {
			return nil, err
		}

		return sets.NewString(config.Spec.API.AccessibleAddons...).List(), nil
	}
}

func ListInstallableAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, configGetter provider.KubermaticConfigurationGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		return handlercommon.ListInstallableAddonEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, configGetter, req.ProjectID, req.ClusterID)
	}
}

func GetAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addonReq)
		return handlercommon.GetAddonEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.AddonID)
	}
}

func ListAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		return handlercommon.ListAddonEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
}

func CreateAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createReq)
		return handlercommon.CreateAddonEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.Body, req.ProjectID, req.ClusterID)
	}
}

func PatchAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchReq)
		return handlercommon.PatchAddonEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.Body, req.ProjectID, req.ClusterID, req.AddonID)
	}
}

func DeleteAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addonReq)
		return handlercommon.DeleteAddonEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.AddonID)
	}
}

func ListAddonConfigsEndpoint(addonConfigProvider provider.AddonConfigProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return handlercommon.ListAddonConfigsEndpoint(addonConfigProvider)
	}
}

func GetAddonConfigEndpoint(addonConfigProvider provider.AddonConfigProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getConfigReq)

		return handlercommon.GetAddonConfigEndpoint(addonConfigProvider, req.AddonID)
	}
}
