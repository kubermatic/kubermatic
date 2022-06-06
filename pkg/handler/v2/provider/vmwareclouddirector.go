/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"github.com/gorilla/mux"

	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// listTemplatesReq defines HTTP request for listing templates.
// swagger:parameters listTemplatesReq
type listTemplatesReq struct {
	cluster.GetClusterReq
	// in: path
	// required: true
	CatalogName string `json:"catalog_name"`
}

// Validate validates listTemplatesReq request.
func (r listTemplatesReq) Validate() error {
	if len(r.CatalogName) == 0 {
		return fmt.Errorf("catalog name cannot be empty")
	}
	return nil
}

func DecodeListTemplatesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listTemplatesReq

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	req.CatalogName = mux.Vars(r)["catalog_name"]
	return req, nil
}

func VMwareCloudDirectorNetworksWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(cluster.GetClusterReq)
		return providercommon.VMwareCloudDirectorNetworksWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, settingsProvider, req.ProjectID, req.ClusterID)
	}
}

func VMwareCloudDirectorTemplatesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listTemplatesReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		return providercommon.VMwareCloudDirectorTemplatesWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, settingsProvider, req.ProjectID, req.ClusterID, req.CatalogName)
	}
}

func VMwareCloudDirectorCatalogsWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(cluster.GetClusterReq)
		return providercommon.VMwareCloudDirectorCatalogsWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, settingsProvider, req.ProjectID, req.ClusterID)
	}
}
