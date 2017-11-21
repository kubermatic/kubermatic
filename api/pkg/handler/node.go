package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	nodesetv1alpha1 "github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

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

// NodeList is an alias for the swagger definition
// swagger:response NodeList
type NodeList = apiv1.NodeList

func nodesEndpoint(kp provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req := request.(nodesReq)

		c, err := kp.Cluster(user, req.Cluster)
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

func deleteNodeEndpoint(kp provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req := request.(nodeReq)

		c, err := kp.Cluster(user, req.Cluster)
		if err != nil {
			return nil, err
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

func createNodesEndpoint(kp provider.ClusterProvider, cps map[string]provider.CloudProvider, dp provider.DataProvider, versions map[string]*api.MasterVersion) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req := request.(createNodesReq)
		c, err := kp.Cluster(user, req.Cluster)
		if err != nil {
			return nil, err
		}

		keys, err := dp.ClusterSSHKeys(user, c.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve ssh keys: %v", err)
		}
		version, found := versions[c.Spec.MasterVersion]
		if !found {
			return nil, fmt.Errorf("unknown cluster version %s", c.Spec.MasterVersion)
		}

		_, cp, err := provider.ClusterCloudProvider(cps, c)
		if err != nil {
			return nil, err
		}
		if cp == nil {
			return nil, errors.NewBadRequest("cannot create nodes without cloud provider")
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
			n.Name = fmt.Sprintf("kubermatic-%s-%s", c.Name, rand.String(5))
			n.Labels = map[string]string{
				LabelArch:     "amd64",
				LabelOS:       "linux",
				LabelHostname: n.Name,
			}

			n.Annotations = map[string]string{
				nodesetv1alpha1.NodeClassNameAnnotationKey: nc.Name,
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
	ClusterReq
}

func decodeNodesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req nodesReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = cr.(ClusterReq)

	return req, nil
}

type createNodesReq struct {
	ClusterReq
	Instances int          `json:"instances"`
	Spec      api.NodeSpec `json:"spec"`
}

func decodeCreateNodesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createNodesReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = cr.(ClusterReq)

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
