package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
)

func datacentersEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		dcs := make([]api.Datacenter, 0, len(kps))
		for dcName, kp := range kps {
			dc := api.Datacenter{
				Metadata: api.Metadata{
					Name:     dcName,
					Revision: "1",
				},
				Spec: *kp.Spec(),
			}
			dcs = append(dcs, dc)
		}

		return dcs, nil
	}
}

func datacenterEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(dcReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewNotFound("kubernetes datacenter", req.dc)
		}

		return &api.Datacenter{
			Metadata: api.Metadata{
				Name:     req.dc,
				Revision: "1",
			},
			Spec: *kp.Spec(),
		}, nil
	}
}

type dcsReq struct {
}

func decodeDatacentersReq(r *http.Request) (interface{}, error) {
	return dcsReq{}, nil
}

func decodeDatacenterReq(r *http.Request) (interface{}, error) {
	return dcReq{
		dc: mux.Vars(r)["dc"],
	}, nil
}
