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
	kerrors "k8s.io/kubernetes/pkg/api/errors"
)

func newClusterEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(newClusterReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.NewCluster(req.user, req.cluster.Metadata.Name, &req.cluster.Spec)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return nil, NewConflict("cluster", req.dc, req.cluster.Spec.HumanReadableName)
			}
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
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		return c, nil
	}
}

func setCloudEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(setCloudReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		if req.provider != "" {
			_, found := cps[req.provider]
			if !found {
				return nil, fmt.Errorf("invalid cloud provider %q", req.provider)
			}

			// TODO(sttts): add cloud credential smoke test
		}

		c, err := kp.SetCloud(req.user, req.cluster, &req.cloud)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
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
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		cs, err := kp.Clusters(req.user)
		if err != nil {
			return nil, err
		}

		return cs, nil
	}
}

func deleteClusterEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteClusterReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		err := kp.DeleteCluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		return struct{}{}, nil
	}
}

type dcReq struct {
	userReq
	dc string
}

func decodeDcReq(r *http.Request) (interface{}, error) {
	var req dcReq

	dr, err := decodeUserReq(r)
	if err != nil {
		return nil, err
	}
	req.userReq = dr.(userReq)

	req.dc = mux.Vars(r)["dc"]
	return req, nil
}

type newClusterReq struct {
	dcReq
	cluster api.Cluster
}

func decodeNewClusterReq(r *http.Request) (interface{}, error) {
	var req newClusterReq

	dr, err := decodeDcReq(r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	if err := json.NewDecoder(r.Body).Decode(&req.cluster); err != nil {
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

type setCloudReq struct {
	clusterReq
	provider string
	cloud    api.CloudSpec
}

func decodeSetCloudReq(r *http.Request) (interface{}, error) {
	var req setCloudReq

	cr, err := decodeClusterReq(r)
	if err != nil {
		return nil, err
	}
	req.clusterReq = cr.(clusterReq)

	if err = json.NewDecoder(r.Body).Decode(&req.cloud); err != nil {
		return nil, err
	}

	req.provider, err = provider.ClusterCloudProviderName(&req.cloud)
	if err != nil {
		return nil, err
	}

	return req, nil
}

type deleteClusterReq struct {
	dcReq
	cluster string
}

func decodeDeleteClusterReq(r *http.Request) (interface{}, error) {
	var req deleteClusterReq

	dr, err := decodeDcReq(r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	req.cluster = mux.Vars(r)["cluster"]

	return req, nil
}
