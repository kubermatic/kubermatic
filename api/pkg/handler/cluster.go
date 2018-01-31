package handler

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

func newClusterEndpoint(kp provider.ClusterProvider, dp provider.SSHKeyProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req := request.(NewClusterReq)

		if req.Body.Cluster == nil {
			return nil, errors.NewBadRequest("no cluster spec given")
		}

		if req.Body.Cluster.Cloud == nil {
			return nil, errors.NewBadRequest("no cloud spec given")
		}

		if req.Body.Cluster.Cloud.DatacenterName == "" && req.Body.Cluster.SeedDatacenterName == "" {
			return nil, errors.NewBadRequest("no datacenter given")
		}

		// As we don't provision byo nodes, we need to allow 0 keys.
		if len(req.Body.SSHKeys) < 1 && req.Body.Cluster.Cloud.BringYourOwn == nil {
			return nil, errors.NewBadRequest("please provide at least one key")
		}

		c, err := kp.NewClusterWithCloud(user, req.Body.Cluster)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return nil, errors.NewConflict("cluster", req.Body.Cluster.Cloud.DatacenterName, req.Body.Cluster.HumanReadableName)
			}
			return nil, err
		}

		err = dp.AssignSSHKeysToCluster(user, req.Body.SSHKeys, c.Name)
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
