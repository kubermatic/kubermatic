package handler

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func getClusterUpgrades(updateManager UpdateManager, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		req, ok := request.(GetClusterReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, GetClusterReq{})
		}

		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		versions, err := updateManager.GetPossibleUpdates(cluster.Spec.Version.String())
		if err != nil {
			return nil, err
		}
		var upgrades []*apiv1.MasterVersion
		for _, v := range versions {
			upgrades = append(upgrades, &apiv1.MasterVersion{
				Version:             v.Version,
				AllowedNodeVersions: v.AllowedNodeVersions,
			})
		}

		return upgrades, nil
	}
}

func getMasterVersions(updateManager UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		versions, err := updateManager.GetMasterVersions()
		if err != nil {
			return nil, fmt.Errorf("failed to get master versions: %v", err)
		}

		sv := make([]*apiv1.MasterVersion, len(versions))
		for v := range versions {
			sv[v] = &apiv1.MasterVersion{
				Version:             versions[v].Version,
				AllowedNodeVersions: versions[v].AllowedNodeVersions,
				Default:             versions[v].Default,
			}
		}

		return sv, nil
	}
}
