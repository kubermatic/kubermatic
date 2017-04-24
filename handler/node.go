package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func nodesEndpointV2(
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

		client, err := c.GetClient()
		if err != nil {
			return nil, err
		}

		cpNodes, err := cp.Nodes(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch nodes from cloud provider: %v", err)
		}
		knodes, err := client.Nodes().List(metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch nodes from apiserver: %v", err)
		}

		getK8sNode := func(nodeName string) (*v1.Node, error) {
			for _, knode := range knodes.Items {
				if nodeName == knode.Name {
					return &knode, nil
				}
			}
			return nil, fmt.Errorf("node %q not found on apiserver", nodeName)
		}

		getNodeCondition := func(n *v1.Node) api.NodeCondition {
			messages := []string{}
			for _, d := range n.Status.Conditions {
				if d.Status != v1.ConditionFalse && d.Type != v1.NodeReady {
					messages = append(messages, d.Message)
				}
			}
			return api.NodeCondition{
				Healthy:     len(messages) == 0,
				Description: strings.Join(messages, ", "),
			}
		}

		for i := range cpNodes {
			k8node, err := getK8sNode(cpNodes[i].Metadata.Name)
			if err == nil {
				cpNodes[i].Status.Versions = &api.NodeVersions{
					ContainerRuntime: k8node.Status.NodeInfo.ContainerRuntimeVersion,
					Kernel:           k8node.Status.NodeInfo.KernelVersion,
					Kubelet:          k8node.Status.NodeInfo.KubeletVersion,
					KubeProxy:        k8node.Status.NodeInfo.KubeProxyVersion,
					OS:               k8node.Status.NodeInfo.OSImage,
				}

				cpNodes[i].Status.CPU = k8node.Status.Allocatable.Cpu().Value()
				cpNodes[i].Status.Memory = k8node.Status.Allocatable.Memory().String()
				cpNodes[i].Status.Condition = getNodeCondition(k8node)
			} else {
				cpNodes[i].Status.Condition = api.NodeCondition{
					Healthy:     false,
					Description: "The node did not joined the cluster so far",
				}
			}
		}

		return cpNodes, nil
	}
}

func kubernetesNodesEndpoint(kps map[string]provider.KubernetesProvider) endpoint.Endpoint {
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

		return client.Nodes().List(metav1.ListOptions{})
	}
}

func kubernetesNodeInfoEndpoint(kps map[string]provider.KubernetesProvider) endpoint.Endpoint {
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

		client, err := c.GetClient()
		if err != nil {
			return nil, err
		}

		return client.Nodes().Get(req.uid, metav1.GetOptions{})
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

		client, err := c.GetClient()
		if err != nil {
			return nil, err
		}

		nodes, err := cp.Nodes(ctx, c)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			if node.Metadata.UID == req.uid {
				err = client.Nodes().Delete(node.Metadata.Name, &metav1.DeleteOptions{})
				if err != nil {
					glog.Errorf("failed to delete node %q from cluster: %v", node.Metadata.Name, err)
				}
			}
		}

		return nil, cp.DeleteNodes(ctx, c, []string{req.uid})
	}
}

func createNodesEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
	masterClientset extensions.Clientset,
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

		nodes, err := cp.CreateNodes(ctx, c, &req.Spec, req.Instances, keys)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			node.Metadata.User = req.user.Name
			_, err = kp.CreateNode(req.user, req.cluster, node)
			if err != nil {
				return nil, err
			}
		}

		return nodes, nil
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
