package admin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// ListAdmissionPluginEndpoint returns admission plugin list
func ListAdmissionPluginEndpoint(userInfoGetter provider.UserInfoGetter, admissionPluginProvider provider.AdmissionPluginsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		admissionPluginList, err := admissionPluginProvider.List(userInfo)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		var resultList []apiv1.AdmissionPlugin
		for _, plugin := range admissionPluginList {
			resultList = append(resultList, convertAdmissionPlugin(plugin))
		}
		return resultList, nil
	}
}

// GetAdmissionPluginEndpoint returns the admission plugin
func GetAdmissionPluginEndpoint(userInfoGetter provider.UserInfoGetter, admissionPluginProvider provider.AdmissionPluginsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(admissionPluginReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		admissionPlugin, err := admissionPluginProvider.Get(userInfo, req.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertAdmissionPlugin(*admissionPlugin), nil
	}
}

// DeleteAdmissionPluginEndpoint deletes the admission plugin
func DeleteAdmissionPluginEndpoint(userInfoGetter provider.UserInfoGetter, admissionPluginProvider provider.AdmissionPluginsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(admissionPluginReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		err = admissionPluginProvider.Delete(userInfo, req.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}

// admissionPluginReq defines HTTP request for getAdmissionPlugin and deleteAdmissionPlugin
// swagger:parameters getAdmissionPlugin deleteAdmissionPlugin
type admissionPluginReq struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

func DecodeAdmissionPluginReq(c context.Context, r *http.Request) (interface{}, error) {
	var req admissionPluginReq
	name := mux.Vars(r)["name"]
	if name == "" {
		return nil, fmt.Errorf("'name' parameter is required but was not provided")
	}
	req.Name = name

	return req, nil
}

func convertAdmissionPlugin(admissionPlugin kubermaticv1.AdmissionPlugin) apiv1.AdmissionPlugin {
	return apiv1.AdmissionPlugin{
		Name:        admissionPlugin.Name,
		Plugin:      admissionPlugin.Spec.PluginName,
		FromVersion: admissionPlugin.Spec.FromVersion,
	}
}
