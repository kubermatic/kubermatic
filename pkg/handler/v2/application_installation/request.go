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

package applicationinstallation

import (
	"context"
	"encoding/json"
	"net/http"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
)

// listApplicationInstallationsReq defines HTTP request for listApplicationInstallations
// swagger:parameters listApplicationInstallations
type listApplicationInstallationsReq struct {
	common.ProjectReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

// createApplicationInstallationReq defines HTTP request for createApplicationInstallations
// swagger:parameters createApplicationInstallation
type createApplicationInstallationReq struct {
	common.ProjectReq

	// in: path
	ClusterID string `json:"cluster_id"`

	// in: body
	// required: true
	Body apiv2.ApplicationInstallation
}

func DecodeListApplicationInstallations(c context.Context, r *http.Request) (interface{}, error) {
	var req listApplicationInstallationsReq

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

func DecodeCreateApplicationInstallation(c context.Context, r *http.Request) (interface{}, error) {
	var req createApplicationInstallationReq

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

func (req listApplicationInstallationsReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func (req createApplicationInstallationReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}
