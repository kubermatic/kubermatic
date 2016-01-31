package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/sttts/kubermatik-api/cloud"
	"github.com/sttts/kubermatik-api/cloud/fake"
	"golang.org/x/net/context"

	httptransport "github.com/go-kit/kit/transport/http"
)

func NewCluster(ctx context.Context) http.Handler {
	return httptransport.NewServer(
		ctx,
		newClusterEndpoint(),
		decodeNewClusterReq,
		encodeNewClusterRes,
	)
}

func Clusters(ctx context.Context) http.Handler {
	return httptransport.NewServer(
		ctx,
		clustersEndpoint(),
		decodeClustersReq,
		encodeNewClusterRes,
	)
}

type clustersReq struct {
	provider  string
	clusterID string
}

func clustersEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clustersReq)

		var p cloud.Provider

		switch req.provider {
		case "fake":
			p = fake.NewProvider()
		default:
			return nil, errors.New("unknown provider")
		}

		return p.Clusters()
	}
}

func decodeClustersReq(r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	provider, okProvider := vars["provider"]
	clusterID := vars["clusterID"]
	if !okProvider {
		return nil, errors.New("invalid clusters request")
	}
	return clustersReq{provider, clusterID}, nil
}

type newClusterReq struct {
	Provider string `json: provider`
	Nodes    int    `json: nodes`
}

func newClusterEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(newClusterReq)

		var (
			p    cloud.Provider
			spec cloud.ClusterSpec
		)

		switch req.Provider {
		case "fake":
			p = fake.NewProvider()
			spec = fake.NewSpec(req.Nodes)
		default:
			return nil, errors.New("unknown provider")
		}

		return p.NewCluster(spec)
	}
}

func decodeNewClusterReq(r *http.Request) (interface{}, error) {
	var req newClusterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

func encodeNewClusterRes(w http.ResponseWriter, response interface{}) (err error) {
	return json.NewEncoder(w).Encode(response)
}
