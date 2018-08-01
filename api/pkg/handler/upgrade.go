package handler

import (
	"context"
	"fmt"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-kit/kit/endpoint"
)

func getClusterUpgrades(updateManager UpdateManager, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)

		req, ok := request.(GetClusterReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, GetClusterReq{})
		}

		project, err := projectProvider.Get(user, req.ProjectID)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		c, err := clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		versions, err := updateManager.GetPossibleUpdates(c.Spec.Version)
		if err != nil {
			return nil, err
		}
		var sv []*apiv1.MasterVersion
		for v := range versions {
			sv = append(sv, &apiv1.MasterVersion{
				Version:             versions[v].Version,
				AllowedNodeVersions: versions[v].AllowedNodeVersions,
			})
		}

		return sv, nil
	}
}

func legacyGetClusterUpgrades(updateManager UpdateManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		req, ok := request.(LegacyGetClusterReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, LegacyGetClusterReq{})
		}

		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.ClusterName)
			}
			return nil, err
		}

		versions, err := updateManager.GetPossibleUpdates(c.Spec.Version)
		if err != nil {
			return nil, err
		}
		var sv []*apiv1.MasterVersion
		for v := range versions {
			sv = append(sv, &apiv1.MasterVersion{
				Version:             versions[v].Version,
				AllowedNodeVersions: versions[v].AllowedNodeVersions,
			})
		}

		return sv, nil
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
