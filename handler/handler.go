package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/sttts/kubermatik-api/cloud"
	"github.com/sttts/kubermatik-api/cloud/fake"
	"golang.org/x/net/context"
)

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

		if _, err := p.NewCluster(spec); err != nil {
			return nil, err
		}

		return nil, nil
	}
}

func decodeNewClusterReq(r *http.Request) (interface{}, error) {
	var req newClusterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

func encodeOK(w http.ResponseWriter, response interface{}) (err error) {
	_, err = fmt.Fprintln(w, `"ok"`)
	return err
}

func NewClusterHandler(ctx context.Context) http.Handler {
	return httptransport.NewServer(
		ctx,
		newClusterEndpoint(),
		decodeNewClusterReq,
		encodeOK,
	)
}
