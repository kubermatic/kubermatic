package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
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

func kubernetesNodesEndpoint(kps map[string]provider.KubernetesProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodesReq)

		// Get dc info
		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		// Get cluster from dc
		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			return nil, err
		}

		// Metalinter is buggy due to a bug see: https://github.com/golang/linter/issues/46
		urlKubernetes := fmt.Sprintf(c.Address.URL + "/api/v1/nodes")
		hCl := &http.Client{}
		hReq, err := http.NewRequest("GET", urlKubernetes, nil)
		hReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Address.Token))
		if err != nil {
			return []byte{}, nil
		}

		hReq = hReq.WithContext(ctx)
		res, err := hCl.Do(hReq)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("bad response, got :%d", res.StatusCode)
		}

		return res.Body, nil
	}
}

func kubernetesNodeInfoEndpoint(kps map[string]provider.KubernetesProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(nodeReq)

		// Get dc info
		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		// Get cluster from dc
		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			return nil, err
		}

		urlKubernetes := fmt.Sprintf(c.Address.URL + fmt.Sprintf("/api/v1/nodes/%s", req.uid))
		hCl := &http.Client{}
		hReq, err := http.NewRequest("GET", urlKubernetes, nil)
		hReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Address.Token))
		if err != nil {
			return []byte{}, nil
		}

		hReq = hReq.WithContext(ctx)
		res, err := hCl.Do(hReq)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("bad response, got :%d", res.StatusCode)
		}

		return res.Body, nil
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

func decodeNodesReq(r *http.Request) (interface{}, error) {
	var req nodesReq

	cr, err := decodeClusterReq(r)
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

func decodeCreateNodesReq(r *http.Request) (interface{}, error) {
	var req createNodesReq

	cr, err := decodeClusterReq(r)
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

func decodeNodeReq(r *http.Request) (interface{}, error) {
	var req nodeReq

	cr, err := decodeNodesReq(r)
	if err != nil {
		return nil, err
	}
	req.nodesReq = cr.(nodesReq)
	req.uid = mux.Vars(r)["node"]

	return req, nil
}
