package handler

import (
	"context"

	"github.com/Masterminds/semver"
	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kversion "github.com/kubermatic/kubermatic/api/pkg/controller/version"
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

		current, err := semver.NewVersion(c.Spec.MasterVersion)
		if err != nil {
			return nil, err
		}

		s := kversion.
			NewUpdatePathSearch(versions, updates, kversion.SemverMatcher{})

		possibleUpdates := []semver.Version{}
		for _, ver := range versions {
			v, err := semver.NewVersion(ver.ID)
			if err != nil {
				continue
			}
			if _, err := s.Search(current.Original(), v.Original()); err != nil {
				continue
			}

			if current.LessThan(v) {
				possibleUpdates = append(possibleUpdates, *v)
			}
		}

		return possibleUpdates, nil
	}
}

func performClusterUpgrade(versions map[string]*apiv1.MasterVersion, updates []apiv1.MasterUpdate) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)

		req, ok := request.(UpgradeReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, UpgradeReq{})
		}

		k, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.ClusterName)
			}
			return nil, err
		}

		_, ok = versions[req.To]
		if !ok {
			return nil, errors.NewUnknownVersion(req.To)
		}

		_, err = kversion.
			NewUpdatePathSearch(versions, updates, kversion.SemverMatcher{}).
			Search(k.Spec.MasterVersion, req.To)
		if err != nil {
			return nil, errors.NewUnknownUpgradePath(k.Spec.MasterVersion, req.To)
		}

		return clusterProvider.InitiateClusterUpgrade(user, req.ClusterName, req.To)
	}
}
