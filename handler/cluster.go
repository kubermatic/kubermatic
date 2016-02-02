package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"

	httptransport "github.com/go-kit/kit/transport/http"
)

func NewCluster(ctx context.Context, p provider.ClusterProvider) http.Handler {
	return httptransport.NewServer(
		ctx,
		newClusterEndpoint(p),
		decodeNewClusterReq,
		encodeJSON,
	)
}

func Cluster(ctx context.Context, p provider.ClusterProvider) http.Handler {
	return httptransport.NewServer(
		ctx,
		clusterEndpoint(p),
		decodeClusterReq,
		encodeJSON,
	)
}

func Clusters(ctx context.Context, p provider.ClusterProvider) http.Handler {
	return httptransport.NewServer(
		ctx,
		clustersEndpoint(p),
		decodeClustersReq,
		encodeJSON,
	)
}

func newClusterEndpoint(p provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(newClusterReq)
		return p.NewCluster(req.name, req.spec)
	}
}

func clusterEndpoint(p provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*clusterReq)
		return p.Cluster(req.dc, req.cluster)
	}
}

func clustersEndpoint(p provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var p provider.ClusterProvider
		req := request.(*clustersReq)
		return p.Clusters(req.dc)
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
	spec provider.ClusterSpec
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

func decodeClusterReq(r *http.Request) (interface{}, error) {
	var req clusterReq

	dr, err := decodeDcReq(r)
	if err != nil {
		return nil, err
	}
	req.dcReq = *dr.(*dcReq)

	req.cluster = mux.Vars(r)["cluster"]

	return req, nil
}
