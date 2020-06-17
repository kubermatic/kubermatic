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

package addon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/apimachinery/pkg/runtime"
	k8sjson "k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
)

// addonReq defines HTTP request for getAddon and deleteAddon
// swagger:parameters getAddon deleteAddon
type addonReq struct {
	common.GetClusterReq
	// in: path
	AddonID string `json:"addon_id"`
}

// listReq defines HTTP request for listAddons and listInstallableAddons endpoints
// swagger:parameters listAddons listInstallableAddons
type listReq struct {
	common.GetClusterReq
}

// createReq defines HTTP request for createAddon endpoint
// swagger:parameters createAddon
type createReq struct {
	common.GetClusterReq
	// in: body
	Body apiv1.Addon
}

// patchReq defines HTTP request for patchAddon endpoint
// swagger:parameters patchAddon
type patchReq struct {
	addonReq
	// in: body
	Body apiv1.Addon
}

// patchReq defines HTTP request for getAddonConfig endpoint
// swagger:parameters getAddonConfig
type getConfigReq struct {
	// in: path
	AddonID string `json:"addon_id"`
}

func DecodeGetAddon(c context.Context, r *http.Request) (interface{}, error) {
	var req addonReq

	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	addonID, err := decodeAddonID(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)
	req.AddonID = addonID

	return req, nil
}

func DecodeListAddons(c context.Context, r *http.Request) (interface{}, error) {
	var req listReq

	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)

	return req, nil
}

func DecodeCreateAddon(c context.Context, r *http.Request) (interface{}, error) {
	var req createReq

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

func DecodePatchAddon(c context.Context, r *http.Request) (interface{}, error) {
	var req patchReq

	gr, err := DecodeGetAddon(c, r)
	if err != nil {
		return nil, err
	}

	req.addonReq = gr.(addonReq)
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func DecodeGetConfig(c context.Context, r *http.Request) (interface{}, error) {
	var req getConfigReq

	addonID, err := decodeAddonID(c, r)
	if err != nil {
		return nil, err
	}

	req.AddonID = addonID

	return req, nil
}

func decodeAddonID(c context.Context, r *http.Request) (string, error) {
	addonID := mux.Vars(r)["addon_id"]
	if addonID == "" {
		return "", fmt.Errorf("'addon_id' parameter is required but was not provided")
	}

	return addonID, nil
}

func ListAccessibleAddons(accessibleAddons sets.String) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return accessibleAddons.UnsortedList(), nil
	}
}

func ListInstallableAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, accessibleAddons sets.String) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		addons, err := listAddons(ctx, userInfoGetter, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		installedAddons := sets.NewString()
		for _, addon := range addons {
			installedAddons.Insert(addon.Name)
		}

		return accessibleAddons.Difference(installedAddons).UnsortedList(), nil
	}
}

func listAddons(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticapiv1.Cluster, projectID string) ([]*kubermaticapiv1.Addon, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		privilegedAddonProvider := ctx.Value(middleware.PrivilegedAddonProviderContextKey).(provider.PrivilegedAddonProvider)
		return privilegedAddonProvider.ListUnsecured(cluster)
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
	return addonProvider.List(userInfo, cluster)
}

func GetAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addonReq)
		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		addon, err := getAddon(ctx, userInfoGetter, cluster, req.ProjectID, req.AddonID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		result, err := convertInternalAddonToExternal(addon)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return result, nil
	}
}

func getAddon(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticapiv1.Cluster, projectID, addonID string) (*kubermaticapiv1.Addon, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		privilegedAddonProvider := ctx.Value(middleware.PrivilegedAddonProviderContextKey).(provider.PrivilegedAddonProvider)
		return privilegedAddonProvider.GetUnsecured(cluster, addonID)
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
	return addonProvider.Get(userInfo, cluster, addonID)
}

func ListAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		addons, err := listAddons(ctx, userInfoGetter, cluster, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		result, err := convertInternalAddonsToExternal(addons)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return result, nil
	}
}

func CreateAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createReq)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		rawVars, err := convertExternalVariablesToInternal(req.Body.Spec.Variables)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		addon, err := createAddon(ctx, userInfoGetter, cluster, rawVars, req.ProjectID, req.Body.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		result, err := convertInternalAddonToExternal(addon)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return result, nil
	}
}

func createAddon(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticapiv1.Cluster, rawVars *runtime.RawExtension, projectID, name string) (*kubermaticapiv1.Addon, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		privilegedAddonProvider := ctx.Value(middleware.PrivilegedAddonProviderContextKey).(provider.PrivilegedAddonProvider)
		return privilegedAddonProvider.NewUnsecured(cluster, name, rawVars)
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
	return addonProvider.New(userInfo, cluster, name, rawVars)

}

func PatchAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchReq)
		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}

		addon, err := getAddon(ctx, userInfoGetter, cluster, req.ProjectID, req.AddonID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		rawVars, err := convertExternalVariablesToInternal(req.Body.Spec.Variables)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		addon.Spec.Variables = *rawVars

		addon, err = updateAddon(ctx, userInfoGetter, cluster, addon, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		result, err := convertInternalAddonToExternal(addon)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return result, nil
	}
}

func updateAddon(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticapiv1.Cluster, addon *kubermaticapiv1.Addon, projectID string) (*kubermaticapiv1.Addon, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		privilegedAddonProvider := ctx.Value(middleware.PrivilegedAddonProviderContextKey).(provider.PrivilegedAddonProvider)
		return privilegedAddonProvider.UpdateUnsecured(cluster, addon)
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
	return addonProvider.Update(userInfo, cluster, addon)
}

func DeleteAddonEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addonReq)
		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, nil)
		if err != nil {
			return nil, err
		}
		return nil, common.KubernetesErrorToHTTPError(deleteAddon(ctx, userInfoGetter, cluster, req.ProjectID, req.AddonID))
	}
}

func deleteAddon(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticapiv1.Cluster, projectID, addonID string) error {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return err
	}
	if adminUserInfo.IsAdmin {
		privilegedAddonProvider := ctx.Value(middleware.PrivilegedAddonProviderContextKey).(provider.PrivilegedAddonProvider)
		return privilegedAddonProvider.DeleteUnsecured(cluster, addonID)
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return err
	}
	addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
	return addonProvider.Delete(userInfo, cluster, addonID)
}

func ListAddonConfigsEndpoint(addonConfigProvider provider.AddonConfigProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		list, err := addonConfigProvider.List()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalAddonConfigsToExternal(list)
	}
}

func GetAddonConfigEndpoint(addonConfigProvider provider.AddonConfigProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getConfigReq)

		addon, err := addonConfigProvider.Get(req.AddonID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalAddonConfigToExternal(addon)
	}
}

func convertInternalAddonToExternal(internalAddon *kubermaticapiv1.Addon) (*apiv1.Addon, error) {
	result := &apiv1.Addon{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internalAddon.Name,
			Name:              internalAddon.Name,
			CreationTimestamp: apiv1.NewTime(internalAddon.CreationTimestamp.Time),
			DeletionTimestamp: func() *apiv1.Time {
				if internalAddon.DeletionTimestamp != nil {
					deletionTimestamp := apiv1.NewTime(internalAddon.DeletionTimestamp.Time)
					return &deletionTimestamp
				}
				return nil
			}(),
		},
		Spec: apiv1.AddonSpec{
			IsDefault: internalAddon.Spec.IsDefault,
		},
	}
	if len(internalAddon.Spec.Variables.Raw) > 0 {
		if err := k8sjson.Unmarshal(internalAddon.Spec.Variables.Raw, &result.Spec.Variables); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func convertInternalAddonsToExternal(internalAddons []*kubermaticapiv1.Addon) ([]*apiv1.Addon, error) {
	result := []*apiv1.Addon{}

	for _, addon := range internalAddons {
		converted, err := convertInternalAddonToExternal(addon)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}

	return result, nil
}

func convertInternalAddonConfigToExternal(internalAddonConfig *kubermaticapiv1.AddonConfig) (*apiv1.AddonConfig, error) {
	return &apiv1.AddonConfig{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internalAddonConfig.Name,
			Name:              internalAddonConfig.Name,
			CreationTimestamp: apiv1.NewTime(internalAddonConfig.CreationTimestamp.Time),
			DeletionTimestamp: func() *apiv1.Time {
				if internalAddonConfig.DeletionTimestamp != nil {
					deletionTimestamp := apiv1.NewTime(internalAddonConfig.DeletionTimestamp.Time)
					return &deletionTimestamp
				}
				return nil
			}(),
		},
		Spec: internalAddonConfig.Spec,
	}, nil
}

func convertInternalAddonConfigsToExternal(internalAddonConfigs *kubermaticapiv1.AddonConfigList) ([]*apiv1.AddonConfig, error) {
	result := []*apiv1.AddonConfig{}

	for _, internalAddonConfig := range internalAddonConfigs.Items {
		converted, err := convertInternalAddonConfigToExternal(&internalAddonConfig)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}

	return result, nil
}

func convertExternalVariablesToInternal(external map[string]interface{}) (*runtime.RawExtension, error) {
	result := &runtime.RawExtension{}
	raw, err := k8sjson.Marshal(external)
	if err != nil {
		return nil, err
	}
	result.Raw = raw
	return result, nil
}
