package handler

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/Masterminds/semver"
	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/kubermatic/api"
	kversion "github.com/kubermatic/kubermatic/api/pkg/controller/version"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

func getClusterUpgrades(
	kp provider.ClusterProvider,
	versions map[string]*api.MasterVersion,
	updates []api.MasterUpdate,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req, ok := request.(clusterReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, clusterReq{})
		}

		c, err := kp.Cluster(user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.cluster)
			}
			return nil, err
		}

		current, err := semver.NewVersion(c.Spec.MasterVersion)
		if err != nil {
			return nil, err
		}

		s := kversion.
			NewUpdatePathSearch(versions, updates, kversion.EqualityMatcher{})

		possibleUpdates := make([]semver.Version, 0)
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

	defer func() {
		if err := r.Body.Close(); err != nil {
			panic(err)
		}
	}()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	v := new(struct {
		To string
	})

	err = json.Unmarshal(b, v)
	if err != nil {
		return nil, err
	}

	req.to = v.To

	return req, nil
}

func performClusterUpgrade(
	kp provider.ClusterProvider,
	versions map[string]*api.MasterVersion,
	updates []api.MasterUpdate,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req, ok := request.(upgradeReq)
		if !ok {
			return nil, errors.NewWrongRequest(request, upgradeReq{})
		}

		k, err := kp.Cluster(user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.cluster)
			}
			return nil, err
		}

		_, ok = versions[req.to]
		if !ok {
			return nil, errors.NewUnknownVersion(req.to)
		}

		_, err = kversion.
			NewUpdatePathSearch(versions, updates, kversion.SemverMatcher{}).
			Search(k.Spec.MasterVersion, req.to)
		if err != nil {
			return nil, errors.NewUnknownUpgradePath(k.Spec.MasterVersion, req.to)
		}

		return kp.InitiateClusterUpgrade(user, req.cluster, req.to)
	}
}
