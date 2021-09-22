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
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

func GetUpgradesEndpoint(configGetter provider.KubermaticConfigurationGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(GetClusterReq)
		if !ok {
			return nil, errors.NewWrongMethod(request, common.GetClusterReq{})
		}
		return handlercommon.GetUpgradesEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, projectProvider, privilegedProjectProvider, configGetter)
	}
}

func UpgradeNodeDeploymentsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(UpgradeNodeDeploymentsReq)
		if !ok {
			return nil, errors.NewWrongMethod(request, common.GetClusterReq{})
		}
		return handlercommon.UpgradeNodeDeploymentsEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, req.Body, projectProvider, privilegedProjectProvider)
	}
}

// UpgradeNodeDeploymentsReq defines HTTP request for upgradeClusterNodeDeploymentsV2 endpoint
// swagger:parameters upgradeClusterNodeDeploymentsV2
type UpgradeNodeDeploymentsReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`

	// in: body
	Body apiv1.MasterVersion
}

// GetSeedCluster returns the SeedCluster object
func (req UpgradeNodeDeploymentsReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeUpgradeNodeDeploymentsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req UpgradeNodeDeploymentsReq
	projectReq, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = projectReq.(common.ProjectReq)
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}
