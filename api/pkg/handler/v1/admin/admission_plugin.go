package admin

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
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

func convertAdmissionPlugin(admissionPlugin kubermaticv1.AdmissionPlugin) apiv1.AdmissionPlugin {
	return apiv1.AdmissionPlugin{
		Name:        admissionPlugin.Name,
		Plugin:      admissionPlugin.Spec.PluginName,
		FromVersion: admissionPlugin.Spec.FromVersion,
	}
}
