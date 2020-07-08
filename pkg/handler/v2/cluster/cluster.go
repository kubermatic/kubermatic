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
	"fmt"
	"net/http"

	"github.com/kubermatic/kubermatic/pkg/handler/middleware"

	"github.com/go-kit/kit/endpoint"
	"github.com/prometheus/client_golang/prometheus"

	apiv1 "github.com/kubermatic/kubermatic/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	handlercommon "github.com/kubermatic/kubermatic/pkg/handler/common"
	"github.com/kubermatic/kubermatic/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
)

func CreateEndpoint(sshKeyProvider provider.SSHKeyProvider, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter,
	initNodeDeploymentFailures *prometheus.CounterVec, eventRecorderProvider provider.EventRecorderProvider, credentialManager provider.PresetProvider,
	exposeStrategy corev1.ServiceType, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider, updateManager common.UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateClusterReq)
		globalSettings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		err = req.Validate(globalSettings.Spec.ClusterTypeOptions, updateManager)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		return handlercommon.CreateEndpoint(ctx, req.ProjectID, req.Body, sshKeyProvider, projectProvider, privilegedProjectProvider, seedsGetter, initNodeDeploymentFailures, eventRecorderProvider, credentialManager, exposeStrategy, userInfoGetter)

	}
}

// CreateClusterReq defines HTTP request for createCluster
// swagger:parameters createClusterV2
type CreateClusterReq struct {
	common.ProjectReq
	// in: body
	Body apiv1.CreateClusterSpec

	// private field for the seed name. Needed for the cluster provider.
	seedName string
}

// GetDC returns the name of the datacenter seed in the request
func (req CreateClusterReq) GetDC() string {
	return req.seedName
}

func DecodeCreateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateClusterReq

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	if len(req.Body.Cluster.Type) == 0 {
		req.Body.Cluster.Type = apiv1.KubernetesClusterType
	}

	seedName, err := findSeedNameForDatacenter(c, req.Body.Cluster.Spec.Cloud.DatacenterName)
	if err != nil {
		return nil, err
	}
	req.seedName = seedName
	return req, nil
}

// Validate validates CreateEndpoint request
func (req CreateClusterReq) Validate(clusterType kubermaticv1.ClusterType, updateManager common.UpdateManager) error {
	if len(req.ProjectID) == 0 {
		return fmt.Errorf("the service account ID cannot be empty")
	}
	return handlercommon.ValidateClusterSpec(clusterType, updateManager, req.Body)
}

func findSeedNameForDatacenter(ctx context.Context, datacenter string) (string, error) {
	seedsGetter, ok := ctx.Value(middleware.SeedsGetterContextKey).(provider.SeedsGetter)
	if !ok {
		return "", fmt.Errorf("seeds getter is not set")
	}
	seeds, err := seedsGetter()
	if err != nil {
		return "", fmt.Errorf("failed to list seeds: %v", err)
	}
	for name, seed := range seeds {
		if _, ok := seed.Spec.Datacenters[datacenter]; ok {
			return name, nil
		}
	}
	return "", fmt.Errorf("can not find seed for datacenter %s", datacenter)
}
