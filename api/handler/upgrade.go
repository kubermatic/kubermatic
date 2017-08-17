package handler

import (
	"context"
	"net/http"
	"sort"

	kversion "github.com/kubermatic/kubermatic/api/controller/version"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	version "github.com/hashicorp/go-version"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/provider"
)

func getClusterUpgrades(
	kps map[string]provider.KubernetesProvider,
	versions map[string]*api.MasterVersion,
	updates []api.MasterUpdate,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(clusterReq)
		if !ok {
			return nil, NewWrongRequest(request, clusterReq{})
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

		s := kversion.
			NewUpdatePathSearch(versions, updates, kversion.EqualityMatcher{})

		possibleUpdates := make(version.Collection, 0)
		for _, ver := range versions {
			if _, err := s.Search(c.Spec.MasterVersion, ver.ID); err != nil {
				continue
			}
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

type upgradeReq struct {
	clusterReq
	to string
}

func decodeUpgradeReq(c context.Context, r *http.Request) (interface{}, error) {
	var req upgradeReq

	dr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.clusterReq = dr.(clusterReq)

	req.to = mux.Vars(r)["to"]

	return req, nil
}

func performClusterUpgrade(
	kps map[string]provider.KubernetesProvider,
	versions map[string]*api.MasterVersion,
	updates []api.MasterUpdate,
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

		k, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		_, ok = versions[req.to]
		if !ok {
			return nil, NewUnknownVersion(req.to)
		}

		_, err = kversion.
			NewUpdatePathSearch(versions, updates, kversion.EqualityMatcher{}).
			Search(k.Spec.MasterVersion, req.to)
		if err != nil {
			return nil, NewUnknownUpgradePath(k.Spec.MasterVersion, req.to)
		}

		return nil, kp.UpgradeCluster(req.user, req.cluster, req.to)
	}
}
