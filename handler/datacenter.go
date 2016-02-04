package handler

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
)

// Datacenters returns a handler delegating to CloudProvider.Nodes.
func Datacenters(
	ctx context.Context,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) http.Handler {
	return httptransport.NewServer(
		ctx,
		datacentersEndpoint(kps, cps),
		decodeDatacentersReq,
		encodeJSON,
	)
}

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
					Revision: 1,
				},
				Spec: *kp.Spec(),
			}
			dcs = append(dcs, dc)
		}

		return dcs, nil
	}
}

type dcsReq struct {
}

func decodeDatacentersReq(r *http.Request) (interface{}, error) {
	return &dcsReq{}, nil
}
