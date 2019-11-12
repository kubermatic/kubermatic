package admin

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// KubermaticSettingsEndpoint returns global settings
func KubermaticSettingsEndpoint(userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		globalSettings, err := settingsProvider.GetGlobalSettings(userInfo)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return globalSettings.Spec, nil
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

		existingGlobalSettings, err := settingsProvider.GetGlobalSettings(userInfo)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		existingGlobalSettingsJSON, err := json.Marshal(existingGlobalSettings.Spec)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode existing settings: %v", err)
		}

		patchedGlobalSettingsJSON, err := jsonpatch.MergePatch(existingGlobalSettingsJSON, req.Patch)
		if err != nil {
			return nil, errors.NewBadRequest("cannot patch global settings: %v", err)
		}
		var patchedGlobalSettings *kubermaticv1.SettingSpec
		err = json.Unmarshal(patchedGlobalSettingsJSON, &patchedGlobalSettings)
		if err != nil {
			return nil, errors.NewBadRequest("cannot decode patched settings: %v", err)
		}

		existingGlobalSettings.Spec = *patchedGlobalSettings
		globalSettings, err := settingsProvider.UpdateGlobalSettings(userInfo, existingGlobalSettings)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return globalSettings.Spec, nil
	}
}

// patchKubermaticSettingsReq defines HTTP request for patchKubermaticSettings endpoint
// swagger:parameters patchKubermaticSettings
type patchKubermaticSettingsReq struct {
	// in: body
	Patch []byte
}

func DecodePatchKubermaticSettingsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchKubermaticSettingsReq
	var err error

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}
