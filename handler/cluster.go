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

	httptransport "github.com/go-kit/kit/transport/http"
)

// NewCluster creates a handler delegating to KubernetesProvider.NewCluster.
func NewCluster(
	ctx context.Context,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) http.Handler {
	return httptransport.NewServer(
		ctx,
		newClusterEndpoint(kps, cps),
		decodeNewClusterReq,
		encodeJSON,
	)
}

// Cluster creates a handler delegating to KubernetesProvider.Cluster.
func Cluster(
	ctx context.Context,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) http.Handler {
	return httptransport.NewServer(
		ctx,
		clusterEndpoint(kps, cps),
		decodeClusterReq,
		encodeJSON,
	)
}

// Clusters creates a handler delegating to KubernetesProvider.Clusters.
func Clusters(
	ctx context.Context,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) http.Handler {
	return httptransport.NewServer(
		ctx,
		clustersEndpoint(kps, cps),
		decodeClustersReq,
		encodeJSON,
	)
}

func newClusterEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(newClusterReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, fmt.Errorf("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.NewCluster(req.name, req.spec)
		if err != nil {
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
			return nil, fmt.Errorf("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.cluster)
		if err != nil {
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
			return nil, fmt.Errorf("unknown kubernetes datacenter %q", req.dc)
		}

		cs, err := kp.Clusters()
		if err != nil {
			return nil, err
		}

		return cs, nil
	}
}

type dcReq struct {
	dc string
}

func decodeDcReq(r *http.Request) (interface{}, error) {
	var req dcReq
	req.dc = mux.Vars(r)["dc"]
	return req, nil
}

type newClusterReq struct {
	dcReq
	name string
	spec api.ClusterSpec
}

func decodeNewClusterReq(r *http.Request) (interface{}, error) {
	var req newClusterReq

	dr, err := decodeDcReq(r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	if err := json.NewDecoder(r.Body).Decode(&req.spec); err != nil {
		return nil, err
	}

	return req, nil
}

type clustersReq struct {
	dcReq
}

func decodeClustersReq(r *http.Request) (interface{}, error) {
	var req clustersReq

	dr, err := decodeDcReq(r)
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

func decodeClusterReq(r *http.Request) (interface{}, error) {
	var req clusterReq

	dr, err := decodeDcReq(r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	req.cluster = mux.Vars(r)["cluster"]

	return req, nil
}
