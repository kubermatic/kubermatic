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
func NewCluster(
	ctx context.Context,
	kp provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) http.Handler {
	return httptransport.NewServer(
		ctx,
		newClusterEndpoint(kp, cps),
		decodeNewClusterReq,
		encodeJSON,
	)
}

// Cluster creates a handler delegating to KubernetesProvider.Cluster.
func Cluster(
	ctx context.Context,
	kp provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) http.Handler {
	return httptransport.NewServer(
		ctx,
		clusterEndpoint(kp, cps),
		decodeReq,
		encodeJSON,
	)
}

// Clusters creates a handler delegating to KubernetesProvider.Clusters.
func Clusters(
	ctx context.Context,
	kp provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) http.Handler {
	return httptransport.NewServer(
		ctx,
		clustersEndpoint(kp, cps),
		decodeClustersReq,
		encodeJSON,
	)
}

func newClusterEndpoint(
	kp provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(newClusterReq)
		c, err := kp.NewCluster(req.name, req.spec)
		if err != nil {
			return nil, err
		}

		err = marshalClusterCloud(cps, c)
		if err != nil {
			return nil, err
		}

		return c, nil
	}
}

func clusterEndpoint(
	kp provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clusterReq)
		c, err := kp.Cluster(req.dc, req.cluster)
		if err != nil {
			return nil, err
		}

		err = unmarshalClusterCloud(cps, c)
		if err != nil {
			return nil, err
		}

		return c, nil
	}
}

func clustersEndpoint(
	kp provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clustersReq)
		cs, err := kp.Clusters(req.dc)
		if err != nil {
			return nil, err
		}

		for _, c := range cs {
			err = unmarshalClusterCloud(cps, c)
			if err != nil {
				return nil, err
			}
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

	if req.spec.Dc != req.dc {
		return nil, errors.New("dc in spec does not match url")
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
