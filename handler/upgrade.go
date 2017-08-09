package handler

import (
	"context"
	"errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/version"
	"github.com/kubermatic/api/provider"
)

type upgradeReq struct {
	clusterReq
}

func getClusterUpgrades(
	kps map[string]provider.KubernetesProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(upgradeReq)

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

		vs, err := version.LoadVersions("")
		if err != nil {
			return nil, err
		}

		possibleUpdates := make([]*api.MasterVersion, 0)

		current := c.Spec.MasterVersion

		for _, v := range vs {
			if v.ID > current {
				possibleUpdates = append(possibleUpdates, v)
			}
		}

		return possibleUpdates, nil
	}
}

func performClusterUpgrade(
	kps map[string]provider.KubernetesProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, errors.New("Not implemented")
	}
}
