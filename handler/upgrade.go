package handler

import (
	"context"
	"sort"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-kit/kit/endpoint"
	version "github.com/hashicorp/go-version"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
)

type upgradeReq struct {
	clusterReq
}

func getClusterUpgrades(
	kps map[string]provider.KubernetesProvider,
	versions map[string]*api.MasterVersion,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(upgradeReq)
		if !ok {
			return nil, NewWrongRequest(request, upgradeReq{})
		}

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		current, err := version.NewVersion(c.Spec.MasterVersion)
		if err != nil {
			return nil, err
		}

		possibleUpdates := make(version.Collection, 0)
		for _, ver := range versions {
			v, err := version.NewVersion(ver.ID)
			if err != nil {
				continue
			}

			if current.LessThan(v) {
				possibleUpdates = append(possibleUpdates, v)
			}
		}
		sort.Sort(possibleUpdates)
		return possibleUpdates, nil
	}
}

func performClusterUpgrade(
	kps map[string]provider.KubernetesProvider,
	updates []api.MasterUpdate,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, NewNotImplemented()
	}
}
