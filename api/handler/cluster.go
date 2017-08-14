package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/extensions"
	"github.com/kubermatic/kubermatic/api/provider"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newClusterEndpointV2(
	kps map[string]provider.KubernetesProvider,
	dcs map[string]provider.DatacenterMeta,
	masterClientset extensions.Clientset,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(newClusterReqV2)

		if req.Cluster == nil {
			return nil, NewBadRequest("no cluster spec given")
		}

		if req.Cluster.Cloud == nil {
			return nil, NewBadRequest("no cloud spec given")
		}

		dc, found := dcs[req.Cluster.Cloud.DatacenterName]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.Cluster.Cloud.DatacenterName)
		}

		kp, found := kps[dc.Seed]
		if !found {
			return nil, NewBadRequest("unknown datacenter %q", dc.Seed)
		}

		if len(req.SSHKeys) < 1 {
			return nil, NewBadRequest("please provide at least one key")
		}

		c, err := kp.NewClusterWithCloud(req.user, req.Cluster)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return nil, NewConflict("cluster", req.Cluster.Cloud.DatacenterName, req.Cluster.HumanReadableName)
			}
			return nil, err
		}

		// TODO(realfake): Duplicated code move to function
		sshClient := masterClientset.SSHKeyTPR(req.user.Name)
		keys, err := sshClient.List()
		if err != nil {
			return nil, err
		}

		for _, key := range keys.Items {
			for _, sshKeyName := range req.SSHKeys {
				if sshKeyName != key.Metadata.Name {
					continue
				}
				key.Clusters = append(key.Clusters, c.Metadata.Name)
				_, err := sshClient.Update(&key)
				if err != nil {
					return nil, err
				}
			}
		}

		return c, nil
	}
}

// Deprecated at V2 of create cluster endpoint
// @TODO Remove with https://github.com/kubermatic/kubermatic/api/issues/220
func newClusterEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(newClusterReq)

		if req.cluster.Spec.Cloud != nil {
			return nil, NewBadRequest("new clusters cannot have a cloud assigned")
		}

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.NewCluster(req.user, &req.cluster.Spec)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return nil, NewConflict("cluster", req.dc, req.cluster.Spec.HumanReadableName)
			}
			return nil, err
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
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		return c, nil
	}
}

// Deprecated at V2 of create cluster endpoint
// @TODO Remove with https://github.com/kubermatic/kubermatic/api/issues/220
func setCloudEndpoint(
	dcs map[string]provider.DatacenterMeta,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(setCloudReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		if req.provider != "" && req.provider != provider.BringYourOwnCloudProvider {
			if _, found := cps[req.provider]; !found {
				return nil, fmt.Errorf("invalid cloud provider %q", req.provider)
			}

			if _, found := dcs[req.cloud.DatacenterName]; !found {
				return nil, fmt.Errorf("invalid node datacenter %q", req.cloud.DatacenterName)
			}

			// TODO(sttts): add cloud credential smoke test
		}

		c, err := kp.SetCloud(req.user, req.cluster, &req.cloud)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
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
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
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
	masterClientset extensions.Clientset,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteClusterReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		//Delete all nodes in the cluster
		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
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
			err = c.Nodes().DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to delete nodes: %v", err)
			}

			for {
				nodes, err := c.Nodes().List(v1.ListOptions{})
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
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		sshClient := masterClientset.SSHKeyTPR(req.user.Name)
		keys, err := sshClient.List()
		if err != nil {
			return nil, err
		}

		for _, key := range keys.Items {
			for i, clusterName := range key.Clusters {
				if clusterName != req.cluster {
					continue
				}
				// TODO(realfake): This takes a long time look forward to async / batch implementation
				key.Clusters = append(key.Clusters[:i], key.Clusters[i+1:]...)
				_, err := sshClient.Update(&key)
				if err != nil {
					return nil, err
				}
			}
		}

		if len(deleteErrors) > 0 {
			err = fmt.Errorf("Please manually clean up any Storage, Nodes or Load Balancers associated with %q, got errors while cleaning up: %v", req.cluster, deleteErrors)
		}

		return nil, err
	}
}

// Deprecated at V2 of create cluster endpoint
// @TODO Remove with https://github.com/kubermatic/kubermatic/api/issues/220
type newClusterReq struct {
	dcReq
	cluster api.Cluster
}

// Deprecated at V2 of create cluster endpoint
// @TODO Remove with https://github.com/kubermatic/kubermatic/api/issues/220
func decodeNewClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req newClusterReq

	dr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	if err := json.NewDecoder(r.Body).Decode(&req.cluster); err != nil {
		return nil, err
	}

	return req, nil
}

type newClusterReqV2 struct {
	userReq
	Cluster *api.ClusterSpec `json:"cluster"`
	SSHKeys []string         `json:"ssh_keys"`
}

func decodeNewClusterReqV2(c context.Context, r *http.Request) (interface{}, error) {
	var req newClusterReqV2

	ur, err := decodeUserReq(r)
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

type setCloudReq struct {
	clusterReq
	provider string
	cloud    api.CloudSpec
}

func decodeSetCloudReq(c context.Context, r *http.Request) (interface{}, error) {
	var req setCloudReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.clusterReq = cr.(clusterReq)

	if err = json.NewDecoder(r.Body).Decode(&req.cloud); err != nil {
		return nil, err
	}

	req.provider, err = provider.ClusterCloudProviderName(&req.cloud)
	if err != nil {
		return nil, err
	}

	if req.provider != "" && req.provider != provider.BringYourOwnCloudProvider &&
		req.cloud.DatacenterName == "" {
		return nil, errors.New("dc cannot be empty when a cloud provider is set")
	}

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
