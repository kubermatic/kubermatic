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
	"errors"
	"fmt"
	"net/http"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/validation/nodeupdate"
	"k8c.io/kubermatic/v2/pkg/version"
)

func GetUpgradesEndpoint(configGetter provider.KubermaticConfigurationGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(common.GetClusterReq)
		if !ok {
			return nil, utilerrors.NewWrongMethod(request, common.GetClusterReq{})
		}
		return handlercommon.GetUpgradesEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, projectProvider, privilegedProjectProvider, configGetter)
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

func GetNodeUpgrades(configGetter provider.KubermaticConfigurationGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(NodeUpgradesReq)
		if !ok {
			return nil, utilerrors.NewWrongMethod(request, NodeUpgradesReq{})
		}
		err := req.TypeReq.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}
		config, err := configGetter(ctx)
		if err != nil {
			return nil, err
		}

		controlPlaneVersion, err := semverlib.NewVersion(req.ControlPlaneVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to parse control plane version: %w", err)
		}

		versions, err := version.NewFromConfiguration(config).GetVersions()
		if err != nil {
			return nil, fmt.Errorf("failed to get master versions: %w", err)
		}

		compatibleVersions, err := filterIncompatibleVersions(versions, controlPlaneVersion)
		if err != nil {
			return nil, fmt.Errorf("failed filter incompatible versions: %w", err)
		}

		return convertVersionsToExternal(compatibleVersions), nil
	}
}

func filterIncompatibleVersions(possibleKubeletVersions []*version.Version, controlPlaneVersion *semverlib.Version) ([]*version.Version, error) {
	var compatibleVersions []*version.Version
	for _, v := range possibleKubeletVersions {
		if err := nodeupdate.EnsureVersionCompatible(controlPlaneVersion, v.Version); err == nil {
			compatibleVersions = append(compatibleVersions, v)
		} else if !errors.Is(err, nodeupdate.VersionSkewError{}) {
			return nil, fmt.Errorf("failed to check compatibility between kubelet %q and control plane %q: %w", v.Version, controlPlaneVersion, err)
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
			return nil, utilerrors.NewWrongMethod(request, common.GetClusterReq{})
		}
		return handlercommon.UpgradeNodeDeploymentsEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, req.Body, projectProvider, privilegedProjectProvider)
	}
}

func GetMasterVersionsEndpoint(configGetter provider.KubermaticConfigurationGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(TypeReq)
		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		config, err := configGetter(ctx)
		if err != nil {
			return nil, err
		}

		versions, err := version.NewFromConfiguration(config).GetVersions()
		if err != nil {
			return nil, fmt.Errorf("failed to get master versions: %w", err)
		}
		return convertVersionsToExternal(versions), nil
	}
}

// TypeReq represents a request that contains the cluster type.
type TypeReq struct {
	// Type is deprecated and not used anymore.
	// in: query
	Type string `json:"type"`
}

func (r TypeReq) Validate() error {
	return nil
}

// DecodeAddReq  decodes an HTTP request into TypeReq.
func DecodeClusterTypeReq(c context.Context, r *http.Request) (interface{}, error) {
	return TypeReq{}, nil
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
