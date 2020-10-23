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

package node

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func DeleteNodeForClusterLegacyEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteNodeForClusterLegacyReq)
		return handlercommon.DeleteMachineNode(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.NodeID)
	}
}

// deleteNodeForClusterLegacyReq defines HTTP request for deleteNodeForClusterLegacy
// swagger:parameters deleteNodeForClusterLegacy
type deleteNodeForClusterLegacyReq struct {
	common.GetClusterReq
	// in: path
	NodeID string `json:"node_id"`
}

func DecodeDeleteNodeForClusterLegacy(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteNodeForClusterLegacyReq

	nodeID := mux.Vars(r)["node_id"]
	if nodeID == "" {
		return "", fmt.Errorf("'node_id' parameter is required but was not provided")
	}

	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID
	req.NodeID = nodeID
	req.DCReq = dcr.(common.DCReq)

	return req, nil
}
