package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/sttts/kubermatic-api/cloud"
	"github.com/sttts/kubermatic-api/cloud/fake"
	"golang.org/x/net/context"

	httptransport "github.com/go-kit/kit/transport/http"
)

func StatusOK(res http.ResponseWriter, _ *http.Request) {
	res.WriteHeader(http.StatusOK)
}

func NewCluster(ctx context.Context) http.Handler {
	return httptransport.NewServer(
		ctx,
		newClusterEndpoint(),
		decodeNewClusterReq,
		encodeJSON,
	)
}

func Clusters(ctx context.Context) http.Handler {
	return httptransport.NewServer(
		ctx,
		clustersEndpoint(),
		decodeClustersReq,
		encodeJSON,
	)
}

func clustersEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var p cloud.Provider

		switch request.(string) {
		case "fake":
			p = fake.NewProvider()
		default:
			return nil, errors.New("unknown provider")
		}

		return p.Clusters()
	}
}

func decodeClustersReq(r *http.Request) (interface{}, error) {
	provider, ok := mux.Vars(r)["provider"]
	if !ok {
		return nil, errors.New("invalid clusters request")
	}
	return provider, nil
}

type newClusterReq struct {
	Provider string
	Nodes    int `json:"nodes"`
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

	req.Provider = mux.Vars(r)["provider"]
	return req, nil
}

func encodeJSON(w http.ResponseWriter, response interface{}) (err error) {
	return json.NewEncoder(w).Encode(response)
}
