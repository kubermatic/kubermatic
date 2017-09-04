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
	"github.com/kubermatic/kubermatic/api/handler/errors"
	crdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/ssh"
	"github.com/kubermatic/kubermatic/api/provider"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

const (
	// NodeDeletionWaitInterval defines how long to wait between the checks if the node has already gone
	NodeDeletionWaitInterval = 500 * time.Millisecond
)

// Copy from k8s.io/kubernetes/pkg/kubelet/apis
const (
	LabelHostname          = "kubernetes.io/hostname"
	LabelZoneFailureDomain = "failure-domain.beta.kubernetes.io/zone"
	LabelZoneRegion        = "failure-domain.beta.kubernetes.io/region"

	LabelInstanceType = "beta.kubernetes.io/instance-type"

	LabelOS   = "beta.kubernetes.io/os"
	LabelArch = "beta.kubernetes.io/arch"
)

func nodesEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodesReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, errors.NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			return nil, err
		}

		client, err := c.GetClient()
		if err != nil {
			return nil, err
		}

		nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
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
			return nil, errors.NewBadRequest("unknown kubernetes datacenter %q", req.dc)
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
			err := clientset.CoreV1().Nodes().Delete(name, &metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			for {
				_, err := clientset.CoreV1().Nodes().Get(name, metav1.GetOptions{})
				if err != nil {
					if apierrors.IsNotFound(err) {
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
	masterClientset crdclient.Interface,
	versions map[string]*api.MasterVersion,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createNodesReq)
		kp, found := kps[req.dc]
		if !found {
			return nil, errors.NewBadRequest("unknown kubernetes datacenter %q", req.dc)
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
			return nil, errors.NewBadRequest("cannot create nodes without cloud provider")
		}

		npName, err := provider.NodeCloudProviderName(&req.Spec)
		if err != nil {
			return nil, err
		}
		if npName != cpName {
			return nil, errors.NewBadRequest("cluster cloud provider %q and node cloud provider %q do not match",
				cpName, npName)
		}

		var keys []v1.UserSSHKey
		opts, err := ssh.UserListOptions(req.user.Name)
		if err != nil {
			return nil, err
		}
		keyList, err := masterClientset.KubermaticV1().UserSSHKeies().List(opts)
		if err != nil {
			return nil, err
		}
		for _, key := range keyList.Items {
			if (&key).IsUsedByCluster(req.cluster) {
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
		if apierrors.IsNotFound(err) {
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
			n := &apiv1.Node{}
			n.Name = fmt.Sprintf("kubermatic-%s-%s", c.Metadata.Name, rand.String(5))
			n.Labels = map[string]string{
				"node.k8s.io/controller": "kube-machine",
				LabelArch:                "amd64",
				LabelOS:                  "linux",
				LabelHostname:            n.Name,
			}
			n.Annotations = map[string]string{
				"node.k8s.io/node-class": nc.Name,
			}

			n, err = client.CoreV1().Nodes().Create(n)
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
