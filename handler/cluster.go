package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"

	httptransport "github.com/go-kit/kit/transport/http"
)

// NewCluster creates a handler delegating to KubernetesProvider.NewCluster.
func NewCluster(ctx context.Context, kp provider.KubernetesProvider) http.Handler {
	return httptransport.NewServer(
		ctx,
		newClusterEndpoint(kp),
		decodeNewClusterReq,
		encodeJSON,
	)
}

// Cluster creates a handler delegating to KubernetesProvider.Cluster.
func Cluster(ctx context.Context, kp provider.KubernetesProvider) http.Handler {
	return httptransport.NewServer(
		ctx,
		clusterEndpoint(kp),
		decodeReq,
		encodeJSON,
	)
}

// Clusters creates a handler delegating to KubernetesProvider.Clusters.
func Clusters(ctx context.Context, kp provider.KubernetesProvider) http.Handler {
	return httptransport.NewServer(
		ctx,
		clustersEndpoint(kp),
		decodeClustersReq,
		encodeJSON,
	)
}

func newClusterEndpoint(kp provider.KubernetesProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(newClusterReq)
		return kp.NewCluster(req.name, req.spec)
	}
}

func clusterEndpoint(kp provider.KubernetesProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*clusterReq)
		return kp.Cluster(req.dc, req.cluster)
	}
}

func clustersEndpoint(kp provider.KubernetesProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*clustersReq)
		return kp.Clusters(req.dc)
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
	req.dcReq = *dr.(*dcReq)

	if err := json.NewDecoder(r.Body).Decode(&req.spec); err != nil {
		return nil, err
	}

	if req.spec.Dc != req.dc {
		return nil, errors.New("dc in spec does not match url")
	}

	return req, nil
}

type clustersReq struct {
	dcReq
}

func decodeClustersReq(r *http.Request) (interface{}, error) {
	return decodeDcReq(r)
}

type clusterReq struct {
	dcReq
	cluster string
}
