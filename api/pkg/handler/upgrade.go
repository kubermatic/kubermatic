package handler

import (
	"context"

	"github.com/Masterminds/semver"
	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

func getClusterUpgrades(versions map[string]*apiv1.MasterVersion, updates []apiv1.MasterUpdate) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)
		req, ok := request.(ClusterReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, ClusterReq{})
		}

		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.ClusterName)
			}
			return nil, err
		}

		var possibleUpdates []semver.Version
		for _, ver := range versions {
			v, err := semver.NewVersion(ver.ID)
			if err != nil {
				continue
			}

			if c.Spec.Version.LessThan(v) {
				possibleUpdates = append(possibleUpdates, *v)
			}
		}

		return possibleUpdates, nil
	}
}
