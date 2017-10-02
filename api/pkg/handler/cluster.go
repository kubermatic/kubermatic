package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/ssh"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		if len(req.SSHKeys) < 1 {
			return nil, errors.NewBadRequest("please provide at least one key")
		}

		c, err := kp.NewClusterWithCloud(user, req.Cluster)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return nil, errors.NewConflict("cluster", req.Cluster.Cloud.DatacenterName, req.Cluster.HumanReadableName)
			}
			return nil, err
		}

		err = dp.AssignSSHKeysToCluster(user.Name, req.SSHKeys, c.Name)
		if err != nil {
			return nil, err
		}

		return c, nil
	}
}

func clusterEndpoint(kp provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req := request.(clusterReq)
		c, err := kp.Cluster(user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.cluster)
			}
			return nil, err
		}

		return c, nil
	}
}

func clustersEndpoint(kp provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req := request.(clustersReq)
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
		req := request.(clusterReq)

		//Delete all nodes in the cluster
		//TODO: Finalizer
		c, err := kp.Cluster(user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.cluster)
			}
			return nil, err
		}

		_, cp, err := provider.ClusterCloudProvider(cps, c)
		if err != nil {
			return nil, err
		}

		var deleteErrors []error
		if cp != nil && c.Status.Phase == kubermaticv1.RunningClusterStatusPhase {
			c, err := c.GetClient()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster client: %v", err)
			}
			err = c.CoreV1().Nodes().DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to delete nodes: %v", err)
			}

			for {
				nodes, err := c.CoreV1().Nodes().List(metav1.ListOptions{})
				if err != nil {
					glog.Errorf("failed to get nodes: %v", err)
					continue
				}
				if len(nodes.Items) == 0 {
					break
				}
				time.Sleep(NodeDeletionWaitInterval)
			}
		}

		err = kp.DeleteCluster(user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				if kerrors.IsNotFound(err) {
					return nil, errors.NewNotFound("cluster", req.cluster)
				}
			}
			return nil, err
		}

		if len(deleteErrors) > 0 {
			err = fmt.Errorf("Please manually clean up any Storage, Nodes or Load Balancers associated with %q, got errors while cleaning up: %v", req.cluster, deleteErrors)
		}

		return nil, err
	}
}

type newClusterReqV2 struct {
	Cluster *kubermaticv1.ClusterSpec `json:"cluster"`
	SSHKeys []string                  `json:"ssh_keys"`
}

func decodeNewClusterReqV2(c context.Context, r *http.Request) (interface{}, error) {
	var req newClusterReqV2
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

type clustersReq struct {
	userReq
}

func decodeClustersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req clustersReq
	dr, err := decodeUserReq(r)
	if err != nil {
		return nil, err
	}
	req.userReq = dr.(userReq)

	return req, nil
}

type clusterReq struct {
	userReq
	cluster string
}

func decodeClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req clusterReq

	dr, err := decodeUserReq(r)
	if err != nil {
		return nil, err
	}
	req.userReq = dr.(userReq)
	req.cluster = mux.Vars(r)["cluster"]

	return req, nil
}
