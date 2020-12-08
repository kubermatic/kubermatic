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

package gatekeeperconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	configv1alpha1 "github.com/open-policy-agent/gatekeeper/apis/config/v1alpha1"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/apimachinery/pkg/types"
)

const (
	// Gatekeeper only uses the configs from the namespace which is set as "gatekeeper namespace" in the gatekeeper controller and audit.
	// For our deployment, its always `gatekeeper-system`
	ConfigNamespace = "gatekeeper-system"
	// Gatekeeper audit also hardcodes the config name to `config`
	ConfigName = "config"
)

func GetEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(gatekeeperConfigReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		clusterCli, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, clus, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		conf := &configv1alpha1.Config{}
		if err := clusterCli.Get(ctx, types.NamespacedName{Namespace: ConfigNamespace, Name: ConfigName}, conf); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiConf, err := convertExternalToAPI(conf)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
		}

		return apiConf, nil
	}
}

// gatekeeperConfigReq defines HTTP request for the gatekeeper sync config
// swagger:parameters getGatekeeperConfig deleteGatekeeperConfig
type gatekeeperConfigReq struct {
	cluster.GetClusterReq
}

func DecodeGatkeeperConfigReq(c context.Context, r *http.Request) (interface{}, error) {
	var req gatekeeperConfigReq

	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(cluster.GetClusterReq)

	return req, nil
}

func convertExternalToAPI(conf *configv1alpha1.Config) (*apiv2.GatekeeperConfig, error) {
	apiConf := &apiv2.GatekeeperConfig{}

	specRaw, err := json.Marshal(&conf.Spec)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling gatekeeper config spec: %v", err)
	}

	err = json.Unmarshal(specRaw, &apiConf.Spec)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling gatekeeper config spec: %v", err)
	}

	return apiConf, nil
}

func DeleteEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(gatekeeperConfigReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		clusterCli, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, clus, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		conf := &configv1alpha1.Config{}
		conf.SetName(ConfigName)
		conf.SetNamespace(ConfigNamespace)
		if err := clusterCli.Delete(ctx, conf); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}
