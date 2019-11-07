package admin

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
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

		return convertInternalSettingsToExternal(globalSettings), nil
	}
}

func convertInternalSettingsToExternal(settings *kubermaticv1.KubermaticSetting) *apiv1.GlobalSettings {
	return &apiv1.GlobalSettings{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                settings.Name,
			Name:              settings.Name,
			DeletionTimestamp: nil,
			CreationTimestamp: apiv1.NewTime(settings.CreationTimestamp.Time),
		},
		Settings: settings.Spec,
	}
}
