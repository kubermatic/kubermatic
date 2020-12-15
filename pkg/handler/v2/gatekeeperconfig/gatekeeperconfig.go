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
	"io/ioutil"
	"net/http"

	jsonpatch "github.com/evanphx/json-patch"
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
	// Gatekeeper audit also uses the hardcoded config name `config`
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
		return nil, fmt.Errorf("error marshalling gatekeeper config spec: %v", err)
	}

	err = json.Unmarshal(specRaw, &apiConf.Spec)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling gatekeeper config spec: %v", err)
	}

	return apiConf, nil
}

func convertAPIToExternal(conf *apiv2.GatekeeperConfig) (*configv1alpha1.Config, error) {
	externalConf := &configv1alpha1.Config{}

	specRaw, err := json.Marshal(&conf.Spec)
	if err != nil {
		return nil, fmt.Errorf("error marshalling gatekeeper config spec: %v", err)
	}

	err = json.Unmarshal(specRaw, &externalConf.Spec)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling gatekeeper config spec: %v", err)
	}

	return externalConf, nil
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

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGatekeeperConfigReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		clusterCli, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, clus, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		conf, err := convertAPIToExternal(&req.Body)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
		}
		conf.SetName(ConfigName)
		conf.SetNamespace(ConfigNamespace)

		if err := clusterCli.Create(ctx, conf); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return req.Body, nil
	}
}

// createGatekeeperConfigReq defines HTTP request for the gatekeeper config create
// swagger:parameters createGatekeeperConfig
type createGatekeeperConfigReq struct {
	cluster.GetClusterReq
	// in: body
	// required: true
	Body apiv2.GatekeeperConfig
}

func DecodeCreateGatkeeperConfigReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createGatekeeperConfigReq

	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(cluster.GetClusterReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

func PatchEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchGatekeeperConfigReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		clus, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		clusterCli, err := common.GetClusterClient(ctx, userInfoGetter, clusterProvider, clus, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		originalConf := &configv1alpha1.Config{}
		if err := clusterCli.Get(ctx, types.NamespacedName{Namespace: ConfigNamespace, Name: ConfigName}, originalConf); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		origninalAPIConfig, err := convertExternalToAPI(originalConf)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
		}

		// patch
		originalJSON, err := json.Marshal(origninalAPIConfig)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("failed to convert current gatkeeper config: %v", err))
		}

		patchedJSON, err := jsonpatch.MergePatch(originalJSON, req.Patch)
		if err != nil {
			return nil, utilerrors.New(http.StatusBadRequest, fmt.Sprintf("failed to merge patch gatkeeper config: %v", err))
		}

		var patched *apiv2.GatekeeperConfig
		err = json.Unmarshal(patchedJSON, &patched)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshall patch gatekeeper config: %v", err))
		}

		patchedGatekeeperConfig, err := convertAPIToExternal(patched)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
		}
		patchedGatekeeperConfig.ObjectMeta = originalConf.ObjectMeta

		if err := clusterCli.Update(ctx, patchedGatekeeperConfig); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return patched, nil
	}
}

// patchGatekeeperConfigReq defines HTTP request for patching gatkeeperconfig
// swagger:parameters patchGatekeeperConfig
type patchGatekeeperConfigReq struct {
	gatekeeperConfigReq
	// in: body
	Patch json.RawMessage
}

// DecodePatchGatekeeperConfigReq decodes http request into patchGatekeeperConfigReq
func DecodePatchGatekeeperConfigReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchGatekeeperConfigReq

	ctReq, err := DecodeGatkeeperConfigReq(c, r)
	if err != nil {
		return nil, err
	}
	req.gatekeeperConfigReq = ctReq.(gatekeeperConfigReq)

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}
