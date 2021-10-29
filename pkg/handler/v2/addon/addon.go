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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

// addonReq defines HTTP request for getAddonV2 and deleteAddonV2
// swagger:parameters getAddonV2 deleteAddonV2
type addonReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: path
	AddonID string `json:"addon_id"`
}

// listReq defines HTTP request for listAddonsV2 and listInstallableAddonsV2 endpoints
// swagger:parameters listAddonsV2 listInstallableAddonsV2
type listReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

// createReq defines HTTP request for createAddon endpoint
// swagger:parameters createAddonV2
type createReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: body
	Body apiv1.Addon
}

// patchReq defines HTTP request for patchAddonV2 endpoint
// swagger:parameters patchAddonV2
type patchReq struct {
	addonReq
	// in: body
	Body apiv1.Addon
}

// GetSeedCluster returns the SeedCluster object
func (req patchReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

// GetSeedCluster returns the SeedCluster object
func (req createReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

// GetSeedCluster returns the SeedCluster object
func (req listReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

// GetSeedCluster returns the SeedCluster object
func (req addonReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeGetAddon(c context.Context, r *http.Request) (interface{}, error) {
	var req addonReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	req.ProjectReq = projectReq.(common.ProjectReq)
	req.ClusterID = clusterID

	addonID, err := decodeAddonID(c, r)
	if err != nil {
		return nil, err
	}
	req.AddonID = addonID

	return req, nil
}

func DecodeListAddons(c context.Context, r *http.Request) (interface{}, error) {
	var req listReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	req.ProjectReq = projectReq.(common.ProjectReq)
	req.ClusterID = clusterID

	return req, nil
}

func DecodeCreateAddon(c context.Context, r *http.Request) (interface{}, error) {
	var req createReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	req.ProjectReq = projectReq.(common.ProjectReq)
	req.ClusterID = clusterID

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

func decodeAddonID(c context.Context, r *http.Request) (string, error) {
	addonID := mux.Vars(r)["addon_id"]
	if addonID == "" {
		return "", fmt.Errorf("'addon_id' parameter is required but was not provided")
	}

	return addonID, nil
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
