package handler

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sort"

	kversion "github.com/kubermatic/kubermatic/api/controller/version"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/blang/semver"
	"github.com/go-kit/kit/endpoint"
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

		current, err := semver.Parse(c.Spec.MasterVersion)
		if err != nil {
			return nil, err
		}

		s := kversion.
			NewUpdatePathSearch(versions, updates, kversion.EqualityMatcher{})

		possibleUpdates := make(semver.Versions, 0)
		for _, ver := range versions {
			if _, err := s.Search(c.Spec.MasterVersion, ver.ID); err != nil {
				continue
			}
			v, err := semver.Parse(ver.ID)
			if err != nil {
				continue
			}

			if current.LT(v) {
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

	type ver struct {
		To string
	}

	dr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.clusterReq = dr.(clusterReq)

	defer func() {
		if err := r.Body.Close(); err != nil {
			panic(err)
		}
	}()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	v := new(ver)

	err = json.Unmarshal(b, v)
	if err != nil {
		return nil, err
	}

	req.to = v.To

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
			NewUpdatePathSearch(versions, updates, kversion.SemverMatcher{}).
			Search(k.Spec.MasterVersion, req.to)
		if err != nil {
			return nil, NewUnknownUpgradePath(k.Spec.MasterVersion, req.to)
		}

		return nil, kp.UpgradeCluster(req.user, req.cluster, req.to)
	}
}
