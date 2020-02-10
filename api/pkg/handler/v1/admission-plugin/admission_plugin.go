package admissionplugin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/apimachinery/pkg/util/sets"
)

func GetAdmissionPluginEndpoint(admissionPluginProvider provider.AdmissionPluginsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(admissionPluginReq)
		pluginResponse, err := admissionPluginProvider.ListPluginNamesFromVersion(req.Version)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// for the backward compatibility we have to keep those plugins as a default
		plugins := sets.NewString(
			"PodSecurityPolicy",
			"PodNodeSelector",
		)
		plugins.Insert(pluginResponse...)

		return plugins.List(), nil
	}
}

// admissionPluginReq defines HTTP request for getAdmissionPlugins
// swagger:parameters getAdmissionPlugins
type admissionPluginReq struct {
	// in: path
	Version string `json:"version"`
}

func DecodeGetAdmissionPlugin(c context.Context, r *http.Request) (interface{}, error) {
	version := mux.Vars(r)["version"]
	if version == "" {
		return nil, fmt.Errorf("'version' parameter is required but was not provided")
	}

	return admissionPluginReq{Version: version}, nil
}
