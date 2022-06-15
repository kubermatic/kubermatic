package resourcequota

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

func GetForProjectEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, quotaProvider provider.ResourceQuotaProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(common.GetProjectRq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if len(req.ProjectID) == 0 {
			return nil, utilerrors.NewBadRequest("the id of the project cannot be empty")
		}

		kubermaticProject, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		userInfo, err := userInfoGetter(ctx, kubermaticProject.Name)
		if err != nil {
			return nil, err
		}

		projectResourceQuota, err := quotaProvider.Get(ctx, userInfo, kubermaticProject.Name, kubermaticProject.Kind)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertResourceQuota(projectResourceQuota), nil
	}
}

func convertResourceQuota(resourceQuota *kubermaticv1.ResourceQuota) *apiv1.ResourceQuota {
	return &apiv1.ResourceQuota{
		Name: resourceQuota.Name,
		Spec: resourceQuota.Spec,
		Status: kubermaticv1.ResourceQuotaStatus{
			LocalUsage: resourceQuota.Status.LocalUsage,
		},
	}
}
