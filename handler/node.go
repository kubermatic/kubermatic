package handler

import (
	"encoding/json"
	"net/http"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	capi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api/v1"
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

		_, cp, err := provider.ClusterCloudProvider(cps, c)
		if err != nil {
			return nil, err
		}
		if cp == nil {
			return []*api.Node{}, nil
		}

		return cp.Nodes(ctx, c)
	}
}

// createClient generates a client from a kube config.
func createClient(ccfg capi.Config) (*kubernetes.Clientset, error) {
	clientConfig := clientcmd.NewNonInteractiveClientConfig(
		ccfg,
		ccfg.CurrentContext,
		&clientcmd.ConfigOverrides{},
		nil,
	)

	cfg, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	c, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func kubeClientFromCluster(dc string, c *api.Cluster) (*kubernetes.Clientset, error) {
	config := getKubeConfig(dc, c)
	client, err := createClient(config)
	return client, err
}

func nodesClientFromDC(
	dc string,
	cluster string,
	user provider.User,
	kps map[string]provider.KubernetesProvider,
) (corev1.NodeInterface, error) {
	// Get dc info
	kp, found := kps[dc]
	if !found {
		return nil, NewBadRequest("unknown kubernetes datacenter %q", dc)
	}

	// Get cluster from dc
	c, err := kp.Cluster(user, cluster)
	if err != nil {
		return nil, err
	}

	client, err := kubeClientFromCluster(dc, c)
	if err != nil {
		return nil, err
	}
	return client.Nodes(), nil
}

func kubernetesNodesEndpoint(kps map[string]provider.KubernetesProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodesReq)

		nodes, err := nodesClientFromDC(req.dc, req.cluster, req.user, kps)
		if err != nil {
			return nil, err
		}
		// TODO(realfake): Which options ?
		return nodes.List(v1.ListOptions{})
	}
}

func kubernetesNodeInfoEndpoint(kps map[string]provider.KubernetesProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodeReq)

		nodes, err := nodesClientFromDC(req.dc, req.cluster, req.user, kps)
		if err != nil {
			return nil, err
		}
		return nodes.Get(req.uid)
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

		return nil, cp.DeleteNodes(ctx, c, []string{req.uid})
	}
}

func createNodesEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
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

		return cp.CreateNodes(ctx, c, &req.Spec, req.Instances)
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
	uid string
}

func decodeNodeReq(c context.Context, r *http.Request) (interface{}, error) {
	var req nodeReq

	cr, err := decodeNodesReq(c, r)
	if err != nil {
		return nil, err
	}
	req.nodesReq = cr.(nodesReq)
	req.uid = mux.Vars(r)["node"]

	return req, nil
}
