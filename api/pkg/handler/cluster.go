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
	"github.com/kubermatic/kubermatic/api"
	crdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/handler/errors"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/ssh"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newClusterEndpointV2(
	kps map[string]provider.KubernetesProvider,
	dcs map[string]provider.DatacenterMeta,
	masterClientset crdclient.Interface,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(newClusterReqV2)

		if req.Cluster == nil {
			return nil, errors.NewBadRequest("no cluster spec given")
		}

		if req.Cluster.Cloud == nil {
			return nil, errors.NewBadRequest("no cloud spec given")
		}

		dc, found := dcs[req.Cluster.Cloud.DatacenterName]
		if !found {
			return nil, errors.NewBadRequest("unknown kubernetes datacenter %q", req.Cluster.Cloud.DatacenterName)
		}

		kp, found := kps[dc.Seed]
		if !found {
			return nil, errors.NewBadRequest("unknown datacenter %q", dc.Seed)
		}

		if len(req.SSHKeys) < 1 {
			return nil, errors.NewBadRequest("please provide at least one key")
		}

		c, err := kp.NewClusterWithCloud(req.user, req.Cluster)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return nil, errors.NewConflict("cluster", req.Cluster.Cloud.DatacenterName, req.Cluster.HumanReadableName)
			}
			return nil, err
		}

		// TODO(realfake): Duplicated code move to function
		opts, err := ssh.UserListOptions(req.user.Name)
		if err != nil {
			return nil, err
		}
		keys, err := masterClientset.KubermaticV1().UserSSHKeies().List(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list ssh-keys: %v", err)
		}

		for _, key := range keys.Items {
			for _, sshKeyName := range req.SSHKeys {
				if sshKeyName != key.ObjectMeta.Name {
					continue
				}
				key.AddToCluster(c.Metadata.Name)
				_, err := masterClientset.KubermaticV1().UserSSHKeies().Update(&key)
				if err != nil {
					return nil, fmt.Errorf("failed to attach ssh-key %q : %v", key.Name, err)
				}
			}
		}

		return c, nil
	}
}

func clusterEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clusterReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, errors.NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		return c, nil
	}
}

func clustersEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clustersReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, errors.NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		cs, err := kp.Clusters(req.user)
		if err != nil {
			return nil, err
		}

		return cs, nil
	}
}

func deleteClusterEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
	masterClientset crdclient.Interface,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteClusterReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, errors.NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		//Delete all nodes in the cluster
		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		_, cp, err := provider.ClusterCloudProvider(cps, c)
		if err != nil {
			return nil, err
		}

		var deleteErrors []error
		if cp != nil && c.Status.Phase == api.RunningClusterStatusPhase {
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

		err = kp.DeleteCluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		opts, err := ssh.UserListOptions(req.user.Name)
		if err != nil {
			return nil, err
		}
		keys, err := masterClientset.KubermaticV1().UserSSHKeies().List(opts)
		if err != nil {
			return nil, err
		}

		for _, key := range keys.Items {
			if !key.IsUsedByCluster(req.cluster) {
				continue
			}
			key.RemoveFromCluster(req.cluster)
			_, err := masterClientset.KubermaticV1().UserSSHKeies().Update(&key)
			if err != nil {
				return nil, err
			}
		}

		if len(deleteErrors) > 0 {
			err = fmt.Errorf("Please manually clean up any Storage, Nodes or Load Balancers associated with %q, got errors while cleaning up: %v", req.cluster, deleteErrors)
		}

		return nil, err
	}
}

type newClusterReqV2 struct {
	userReq
	Cluster *api.ClusterSpec `json:"cluster"`
	SSHKeys []string         `json:"ssh_keys"`
}

func decodeNewClusterReqV2(c context.Context, r *http.Request) (interface{}, error) {
	var req newClusterReqV2

	ur, err := decodeUserReq(c)
	if err != nil {
		return nil, err
	}
	req.userReq = ur.(userReq)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

type clustersReq struct {
	dcReq
}

func decodeClustersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req clustersReq

	dr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	return req, nil
}

type clusterReq struct {
	dcReq
	cluster string
}

func decodeClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req clusterReq

	dr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	req.cluster = mux.Vars(r)["cluster"]

	return req, nil
}

type deleteClusterReq struct {
	dcReq
	cluster string
}

func decodeDeleteClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteClusterReq

	dr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	req.cluster = mux.Vars(r)["cluster"]

	return req, nil
}
