package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// UpdateAdmissionPluginEndpoint updates the admission plugin
func UpdateAdmissionPluginEndpoint(userInfoGetter provider.UserInfoGetter, admissionPluginProvider provider.AdmissionPluginsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateAdmissionPluginReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, k8cerrors.NewBadRequest(err.Error())
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		editedAdmissionPlugin, err := admissionPluginProvider.Update(userInfo, &kubermaticv1.AdmissionPlugin{
			ObjectMeta: v1.ObjectMeta{Name: req.Name},
			Spec: kubermaticv1.AdmissionPluginSpec{
				PluginName:  req.Body.Plugin,
				FromVersion: req.Body.FromVersion,
			},
		})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertAdmissionPlugin(*editedAdmissionPlugin), nil
	}
}

// admissionPluginReq defines HTTP request for getAdmissionPlugin and deleteAdmissionPlugin
// swagger:parameters getAdmissionPlugin deleteAdmissionPlugin
type admissionPluginReq struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

// updateAdmissionPlugin defines HTTP request for updateAdmissionPlugin
// swagger:parameters updateAdmissionPlugin
type updateAdmissionPluginReq struct {
	admissionPluginReq
	// in: body
	Body apiv1.AdmissionPlugin
}

// Validate validates UpdateAdmissionPluginEndpoint request
func (r updateAdmissionPluginReq) Validate() error {
	if r.Name != r.Body.Name {
		return fmt.Errorf("admission plugin name mismatch, you requested to update AdmissionPlugin = %s but body contains AdmissionPlugin = %s", r.Name, r.Body.Name)
	}
	return nil
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

func DecodeUpdateAdmissionPluginReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateAdmissionPluginReq
	pluginReq, err := DecodeAdmissionPluginReq(c, r)
	if err != nil {
		return nil, err
	}
	req.admissionPluginReq = pluginReq.(admissionPluginReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func convertAdmissionPlugin(admissionPlugin kubermaticv1.AdmissionPlugin) apiv1.AdmissionPlugin {
	return apiv1.AdmissionPlugin{
		Name:        admissionPlugin.Name,
		Plugin:      admissionPlugin.Spec.PluginName,
		FromVersion: admissionPlugin.Spec.FromVersion,
	}
}
