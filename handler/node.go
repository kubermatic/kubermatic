package handler

import (
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
		req := request.(*nodesReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.cluster)
		if err != nil {
			return nil, err
		}

		cp, err := provider.ClusterCloudProvider(cps, c)
		if err != nil {
			return nil, err
		}
		if cp == nil {
			return []*api.Node{}, nil
		}

		return cp.Nodes(c)
	}
}

type nodesReq struct {
	dcReq
	cluster string
}

func decodeNodesReq(r *http.Request) (interface{}, error) {
	var req nodesReq

	dr, err := decodeDcReq(r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	req.cluster = mux.Vars(r)["cluster"]

	return req, nil
}
