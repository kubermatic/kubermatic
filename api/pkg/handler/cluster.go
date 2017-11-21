package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

func newClusterEndpointV2(kp provider.ClusterProvider, dp provider.DataProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req := request.(newClusterReqV2)

		if req.Cluster == nil {
			return nil, errors.NewBadRequest("no cluster spec given")
		}

		if req.Cluster.Cloud == nil {
			return nil, errors.NewBadRequest("no cloud spec given")
		}

		if req.Cluster.Cloud.DatacenterName == "" && req.Cluster.SeedDatacenterName == "" {
			return nil, errors.NewBadRequest("no datacenter given")
		}

		// As we don't provision byo nodes, we need to allow 0 keys.
		if len(req.SSHKeys) < 1 && req.Cluster.Cloud.BringYourOwn == nil {
			return nil, errors.NewBadRequest("please provide at least one key")
		}

		c, err := kp.NewClusterWithCloud(user, req.Cluster)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return nil, errors.NewConflict("cluster", req.Cluster.Cloud.DatacenterName, req.Cluster.HumanReadableName)
			}
			return nil, err
		}

		err = dp.AssignSSHKeysToCluster(user, req.SSHKeys, c.Name)
		if err != nil {
			return nil, err
		}

		return c, nil
	}
}

func clusterEndpoint(kp provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req := request.(ClusterReq)
		c, err := kp.Cluster(user, req.Cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.Cluster)
			}
			return nil, err
		}

		return c, nil
	}
}

func clustersEndpoint(kp provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		cs, err := kp.Clusters(user)
		if err != nil {
			return nil, err
		}

		return cs, nil
	}
}

func deleteClusterEndpoint(
	kp provider.ClusterProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req := request.(ClusterReq)

		//Delete all nodes in the cluster
		c, err := kp.Cluster(user, req.Cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.Cluster)
			}
			return nil, err
		}

		return nil, kp.DeleteCluster(user, c.Name)
	}
}

type newClusterReqV2 struct {
	Cluster *kubermaticv1.ClusterSpec `json:"cluster"`
	SSHKeys []string                  `json:"sshKeys"`
}

func decodeNewClusterReqV2(c context.Context, r *http.Request) (interface{}, error) {
	var req newClusterReqV2
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

type clustersReq struct{}

func decodeClustersReq(c context.Context, r *http.Request) (interface{}, error) {
	return clustersReq{}, nil
}

// swagger:parameters performClusterUpgrage
type ClusterReq struct {
	// in: path
	Cluster string
}

func decodeClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClusterReq
	req.Cluster = mux.Vars(r)["cluster"]
	return req, nil
}
