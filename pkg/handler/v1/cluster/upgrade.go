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

	"github.com/Masterminds/semver/v3"
	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/validation/nodeupdate"
	"k8c.io/kubermatic/v2/pkg/version"
)

func GetUpgradesEndpoint(updateManager common.UpdateManager, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(common.GetClusterReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, common.GetClusterReq{})
		}
		return handlercommon.GetUpgradesEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, projectProvider, privilegedProjectProvider, updateManager)
	}
}

// NodeUpgradesReq defines HTTP request for getNodeUpgrades
// swagger:parameters getNodeUpgrades
type NodeUpgradesReq struct {
	TypeReq
	// in: query
	ControlPlaneVersion string `json:"control_plane_version,omitempty"`
}

func DecodeNodeUpgradesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NodeUpgradesReq

	clusterTypeReq, err := DecodeClusterTypeReq(c, r)
	if err != nil {
		return nil, err
	}
	req.TypeReq = clusterTypeReq.(TypeReq)

	req.ControlPlaneVersion = r.URL.Query().Get("control_plane_version")

	return req, nil
}

func GetNodeUpgrades(updateManager common.UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(NodeUpgradesReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, NodeUpgradesReq{})
		}
		err := req.TypeReq.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		controlPlaneVersion, err := semver.NewVersion(req.ControlPlaneVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to parse control plane version: %v", err)
		}

		versions, err := updateManager.GetVersions(req.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get master versions: %v", err)
		}

		compatibleVersions, err := filterIncompatibleVersions(versions, controlPlaneVersion)
		if err != nil {
			return nil, fmt.Errorf("failed filter incompatible versions: %v", err)
		}

		return convertVersionsToExternal(compatibleVersions), nil
	}
}

func filterIncompatibleVersions(possibleKubeletVersions []*version.Version, controlPlaneVersion *semver.Version) ([]*version.Version, error) {
	var compatibleVersions []*version.Version
	for _, v := range possibleKubeletVersions {
		if err := nodeupdate.EnsureVersionCompatible(controlPlaneVersion, v.Version); err == nil {
			compatibleVersions = append(compatibleVersions, v)
		} else {
			_, ok := err.(nodeupdate.ErrVersionSkew)
			if !ok {
				return nil, fmt.Errorf("failed to check compatibility between kubelet %q and control plane %q: %v", v.Version, controlPlaneVersion, err)
			}
		}
	}
	return compatibleVersions, nil
}

// UpgradeNodeDeploymentsReq defines HTTP request for upgradeClusterNodeDeployments endpoint
// swagger:parameters upgradeClusterNodeDeployments
type UpgradeNodeDeploymentsReq struct {
	common.GetClusterReq

	// in: body
	Body apiv1.MasterVersion
}

func DecodeUpgradeNodeDeploymentsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req UpgradeNodeDeploymentsReq
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

func UpgradeNodeDeploymentsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(UpgradeNodeDeploymentsReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, common.GetClusterReq{})
		}
		return handlercommon.UpgradeNodeDeploymentsEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, req.Body, projectProvider, privilegedProjectProvider)
	}
}

func GetMasterVersionsEndpoint(updateManager common.UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(TypeReq)
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		versions, err := updateManager.GetVersions(req.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get master versions: %v", err)
		}
		return convertVersionsToExternal(versions), nil
	}
}

// TypeReq represents a request that contains the cluster type
type TypeReq struct {
	// in: query
	Type string `json:"type"`
}

func (r TypeReq) Validate() error {
	if handlercommon.ClusterTypes.Has(r.Type) {
		return nil
	}
	return fmt.Errorf("invalid cluster type %s", r.Type)
}

// DecodeAddReq  decodes an HTTP request into TypeReq
func DecodeClusterTypeReq(c context.Context, r *http.Request) (interface{}, error) {
	var req TypeReq

	req.Type = r.URL.Query().Get("type")
	if len(req.Type) == 0 {
		req.Type = apiv1.KubernetesClusterType
	}

	return req, nil
}

func convertVersionsToExternal(versions []*version.Version) []*apiv1.MasterVersion {
	sv := make([]*apiv1.MasterVersion, len(versions))
	for v := range versions {
		sv[v] = &apiv1.MasterVersion{
			Version: versions[v].Version,
			Default: versions[v].Default,
		}
	}
	return sv
}
