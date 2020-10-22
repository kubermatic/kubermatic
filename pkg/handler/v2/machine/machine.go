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

package machine

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func CreateMachineDeployment(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createMachineDeploymentReq)
		return handlercommon.CreateMachineDeployment(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, sshKeyProvider, seedsGetter, req.Body, req.ProjectID, req.ClusterID)
	}
}

// createMachineDeploymentReq defines HTTP request for createMachineDeployment
// swagger:parameters createMachineDeployment
type createMachineDeploymentReq struct {
	common.ProjectReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// in: body
	Body apiv1.NodeDeployment
}

func DecodeCreateMachineDeployment(c context.Context, r *http.Request) (interface{}, error) {
	var req createMachineDeploymentReq

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

	if err = json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// GetSeedCluster returns the SeedCluster object
func (req createMachineDeploymentReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}
