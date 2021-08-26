package version

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

func GetUpgradesEndpoint(updateManager common.UpdateManager, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		req, ok := request.(GetClusterReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, common.GetClusterReq{})
		}
		return handlercommon.GetUpgradesEndpoint(ctx, userInfoGetter, req.ProjectID, req.ClusterID, projectProvider, privilegedProjectProvider, updateManager)
	}
}

func ListAvailableVersions(r.updateManager, r.projectProvider, r.privilegedProjectProvider, r.userInfoGetter)
