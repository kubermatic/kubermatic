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

package cluster

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func ListClusterRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		return handlercommon.ListClusterRoleEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
	}
}

func ListClusterRoleNamesEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		return handlercommon.ListClusterRoleNamesEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
	}
}

func ListRoleEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		return handlercommon.ListRoleEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
	}
}

func ListRoleNamesEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		return handlercommon.ListRoleNamesEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID)
	}
}

// listReq defines HTTP request for listClusterRole and listRole endpoint
// swagger:parameters listClusterRoleV2 listRoleV2 listRoleNamesV2 listClusterRoleNamesV2
type listReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
}

func DecodeListClusterRoleReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)
	return req, nil
}

// GetSeedCluster returns the SeedCluster object
func (req listReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}
