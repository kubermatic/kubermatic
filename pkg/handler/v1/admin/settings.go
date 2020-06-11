package admin

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"

	v1 "github.com/kubermatic/kubermatic/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/util/errors"
)

// KubermaticSettingsEndpoint returns global settings
func KubermaticSettingsEndpoint(settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		globalSettings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return v1.GlobalSettings(globalSettings.Spec), nil
	}
}

// KubermaticCustomLinksEndpoint returns custom links
func KubermaticCustomLinksEndpoint(settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		globalSettings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return v1.GlobalCustomLinks(globalSettings.Spec.CustomLinks), nil
	}
}

// UpdateKubermaticSettingsEndpoint updates global settings
func UpdateKubermaticSettingsEndpoint(userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchKubermaticSettingsReq)
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		existingGlobalSettings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		existingGlobalSettingsSpecJSON, err := json.Marshal(existingGlobalSettings.Spec)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode existing settings: %v", err)
		}

		patchedGlobalSettingsSpecJSON, err := jsonpatch.MergePatch(existingGlobalSettingsSpecJSON, req.Patch)
		if err != nil {
			return nil, errors.NewBadRequest("cannot patch global settings: %v", err)
		}
		var patchedGlobalSettingsSpec *kubermaticv1.SettingSpec
		err = json.Unmarshal(patchedGlobalSettingsSpecJSON, &patchedGlobalSettingsSpec)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode patched settings: %v", err)
		}

		existingGlobalSettings.Spec = *patchedGlobalSettingsSpec
		globalSettings, err := settingsProvider.UpdateGlobalSettings(userInfo, existingGlobalSettings)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return v1.GlobalSettings(globalSettings.Spec), nil
	}
}

// patchKubermaticSettingsReq defines HTTP request for patchKubermaticSettings endpoint
// swagger:parameters patchKubermaticSettings
type patchKubermaticSettingsReq struct {
	// in: body
	Patch json.RawMessage
}

func DecodePatchKubermaticSettingsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchKubermaticSettingsReq
	var err error

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}
