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

package version

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	crdapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

// listProviderVersionsReq represents a request for a list of presets
// swagger:parameters listProviderVersionsReq
type listProviderVersionsReq struct {
	// in: path
	// required: true
	ProviderName string `json:"provider_name"`
	// in: query
	Type string `json:"type"`
}

// Validate validates listProviderVersionsReq request
func (l listProviderVersionsReq) Validate() error {
	if len(l.ProviderName) == 0 {
		return fmt.Errorf("the provider name cannot be empty")
	}
	if len(l.Type) == 0 {
		return fmt.Errorf("the type field cannot be empty")
	}
	if !crdapiv1.IsProviderSupported(l.ProviderName) {
		return fmt.Errorf("invalid provider name %s", l.ProviderName)
	}

	return nil
}

func DecodeListProviderVersions(ctx context.Context, r *http.Request) (interface{}, error) {
	clusterType := r.URL.Query().Get("type")
	if len(clusterType) == 0 {
		clusterType = apiv1.KubernetesClusterType
	}

	return listProviderVersionsReq{
		ProviderName: mux.Vars(r)["provider_name"],
		Type:         clusterType,
	}, nil
}

// ListVersions returns a list of available Kubernetes version for the given provider
func ListVersions(updateManager common.UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listProviderVersionsReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		versions, err := updateManager.GetVersionsV2(req.Type, kubermaticv1.ProviderType(req.ProviderName))
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		masterVersions := make([]*apiv1.MasterVersion, len(versions))
		for v := range versions {
			masterVersions[v] = &apiv1.MasterVersion{
				Version: versions[v].Version,
				Default: versions[v].Default,
			}
		}

		return masterVersions, nil
	}
}
