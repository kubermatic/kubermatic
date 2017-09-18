package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/extensions"
	"github.com/kubermatic/kubermatic/api/provider"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	// NodeDeletionWaitInterval defines how long to wait between the checks if the node has already gone
	NodeDeletionWaitInterval = 500 * time.Millisecond
)

func nodesEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodesReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			return nil, err
		}

		client, err := c.GetClient()
		if err != nil {
			return nil, err
		}

		nodes, err := client.Nodes().List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		return nodes.Items, err
	}
}

func deleteNodeEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodeReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			return nil, err
		}

		_, cp, err := provider.ClusterCloudProvider(cps, c)
		if err != nil {
			return nil, err
		}

		if cp == nil {
			return []*api.Node{}, nil
		}

		deleteNodeLocking := func(clientset *kubernetes.Clientset, name string) error {
			err := clientset.Nodes().Delete(name, &metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			for {
				_, err := clientset.Nodes().Get(name, metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						return nil
					}

					return fmt.Errorf("failed to get nodes: %v", err)
				}
				time.Sleep(NodeDeletionWaitInterval)
			}
		}

		client, err := c.GetClient()
		if err != nil {
			return nil, fmt.Errorf("failed to get client: %v", err)
		}
		return nil, deleteNodeLocking(client, req.nodeName)
	}
}

func createNodesEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
	masterClientset extensions.Clientset,
	versions map[string]*api.MasterVersion,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createNodesReq)
		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			return nil, err
		}

		cpName, cp, err := provider.ClusterCloudProvider(cps, c)
		if err != nil {
			return nil, err
		}
		if cp == nil {
			return nil, NewBadRequest("cannot create nodes without cloud provider")
		}

		npName, err := provider.NodeCloudProviderName(&req.Spec)
		if err != nil {
			return nil, err
		}
		if npName != cpName {
			return nil, NewBadRequest("cluster cloud provider %q and node cloud provider %q do not match",
				cpName, npName)
		}

		var keys []extensions.UserSSHKey
		keyList, err := masterClientset.SSHKeyTPR(req.user.Name).List()
		if err != nil {
			return nil, err
		}
		for _, key := range keyList.Items {
			if (&key).UsedByCluster(req.cluster) {
				keys = append(keys, key)
			}
		}

		version, found := versions[c.Spec.MasterVersion]
		if !found {
			return nil, fmt.Errorf("unknown cluster version %s", c.Spec.MasterVersion)
		}

		nclient, err := c.GetNodesetClient()
		if err != nil {
			return nil, err
		}
		nc, err := nclient.NodesetV1alpha1().NodeClasses().Get(cp.GetNodeClassName(&req.Spec), metav1.GetOptions{})
		if errors.IsNotFound(err) {
			nc, err = cp.CreateNodeClass(c, &req.Spec, keys, version)
			if err != nil {
				return nil, err
			}
		}

		client, err := c.GetClient()
		if err != nil {
			return nil, err
		}

		for i := 1; i <= req.Instances; i++ {
			n := &v1.Node{}
			n.Name = fmt.Sprintf("kubermatic-%s-%s", c.Metadata.Name, rand.String(5))
			n.Labels = map[string]string{
				"node.k8s.io/controller": "kube-machine",
				metav1.LabelArch:         "amd64",
				metav1.LabelOS:           "linux",
				metav1.LabelHostname:     n.Name,
			}
			n.Annotations = map[string]string{
				"node.k8s.io/node-class": nc.Name,
			}

			n, err = client.Nodes().Create(n)
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
}

type nodesReq struct {
	clusterReq
}

func decodeNodesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req nodesReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.clusterReq = cr.(clusterReq)

	return req, nil
}

type createNodesReq struct {
	clusterReq
	Instances int          `json:"instances"`
	Spec      api.NodeSpec `json:"spec"`
}

func decodeCreateNodesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createNodesReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.clusterReq = cr.(clusterReq)

	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

type nodeReq struct {
	nodesReq
	nodeName string
}

func decodeNodeReq(c context.Context, r *http.Request) (interface{}, error) {
	var req nodeReq

	cr, err := decodeNodesReq(c, r)
	if err != nil {
		return nil, err
	}
	req.nodesReq = cr.(nodesReq)
	req.nodeName = mux.Vars(r)["node"]

	return req, nil
}
