/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package cniversion

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	v2 "k8c.io/kubermatic/v2/pkg/api/v2"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/version/cni"
)

// ListVersions returns a list of available versions for the given CNI plugin type
func ListVersions() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listCNIPluginVersionsReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		versions, err := cni.GetSupportedCNIPluginVersions(v1.CNIPluginType(req.CNIPluginType))
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		return v2.CNIVersions{
			CNIPluginType: req.CNIPluginType,
			Versions:      versions.List(),
		}, nil
	}
}

// listCNIPluginVersionsReq represents a request for a list of versions for a given cni plugin
// swagger:parameters listVersionsByCNIPlugin
type listCNIPluginVersionsReq struct {
	// in: path
	// required: true
	CNIPluginType string `json:"cni_plugin_type"`
}

func DecodeListCNIPluginVersions(ctx context.Context, r *http.Request) (interface{}, error) {
	cpt, ok := mux.Vars(r)["cni_plugin_type"]
	if !ok {
		return nil, fmt.Errorf("'cni_plugin_type' parameter is required")
	}

	return listCNIPluginVersionsReq{
		CNIPluginType: cpt,
	}, nil
}

// Validate validates listProviderVersionsReq request
func (l listCNIPluginVersionsReq) Validate() error {
	if !cni.GetSupportedCNIPlugins().Has(l.CNIPluginType) {
		return fmt.Errorf("CNI plugin type %q not supported. Supported types: %v", l.CNIPluginType, cni.GetSupportedCNIPlugins().List())
	}
	return nil
}

// ListVersionsForCluster returns a list of available versions for the given CNI plugin type
func ListVersionsForCluster(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listCNIPluginVersionsForClusterReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		c, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		versions, err := cni.GetSupportedCNIPluginVersions(c.Spec.CNIPlugin.Type)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		return v2.CNIVersions{
			CNIPluginType: c.Spec.CNIPlugin.Type.String(),
			Versions:      versions.List(),
		}, nil
	}
}

// listCNIPluginVersionsForClusterReq represents a request for a list of possible cni versions for a cluster
// swagger:parameters listCNIPluginVersionsForCluster
type listCNIPluginVersionsForClusterReq struct {
	cluster.GetClusterReq
}

func DecodeListCNIPluginVersionsForClusterReq(ctx context.Context, r *http.Request) (interface{}, error) {
	var req listCNIPluginVersionsForClusterReq
	cr, err := cluster.DecodeGetClusterReq(ctx, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)
	return req, nil
}
