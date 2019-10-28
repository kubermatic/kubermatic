package addon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

// listReq defines HTTP request for listAddons endpoint
// swagger:parameters listAddons
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

func GetAddonEndpoint(projectProvider provider.ProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addonReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, err = projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
		addon, err := addonProvider.Get(userInfo, cluster, req.AddonID)
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

func ListAddonEndpoint(projectProvider provider.ProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, err = projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
		addons, err := addonProvider.List(userInfo, cluster)
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

func CreateAddonEndpoint(projectProvider provider.ProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, err = projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
		rawVars, err := convertExternalVariablesToInternal(req.Body.Spec.Variables)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		addon, err := addonProvider.New(userInfo, cluster, req.Body.Name, rawVars)
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

func PatchAddonEndpoint(projectProvider provider.ProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, err = projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
		addon, err := addonProvider.Get(userInfo, cluster, req.AddonID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		rawVars, err := convertExternalVariablesToInternal(req.Body.Spec.Variables)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		addon.Spec.Variables = *rawVars

		addon, err = addonProvider.Update(userInfo, cluster, addon)
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

func DeleteAddonEndpoint(projectProvider provider.ProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addonReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, err = projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)

		return nil, common.KubernetesErrorToHTTPError(addonProvider.Delete(userInfo, cluster, req.AddonID))
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

func convertExternalVariablesToInternal(external map[string]interface{}) (*runtime.RawExtension, error) {
	result := &runtime.RawExtension{}
	raw, err := k8sjson.Marshal(external)
	if err != nil {
		return nil, err
	}
	result.Raw = raw
	return result, nil
}
