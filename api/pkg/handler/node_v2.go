package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

// CreateNodeReqV2 represent a request for specific data to create a node
// swagger:parameters createClusterNodeV2
type CreateNodeReqV2 struct {
	ClusterReq
	// in: body
	Body CreateNodeReqBodyV2
}

// CreateNodeReqBodyV2 represents the request body of a create nodes request
type CreateNodeReqBodyV2 struct {
	apiv2.Node
}

// NodeReqV2 represent a request for node specific data
// swagger:parameters getClusterNodeV2 deleteClusterNodeV2
type NodeReqV2 struct {
	ClusterReq
	// in: path
	NodeName string `json:"node"`
}

func decodeCreateNodeReqV2(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateNodeReqV2

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = cr.(ClusterReq)

	if err = json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func createNodeEndpointV2(kp provider.ClusterProvider, cps map[string]provider.CloudProvider, dp provider.SSHKeyProvider, versions map[string]*apiv1.MasterVersion) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, nil
	}
}

func getNodesEndpointV2(kp provider.ClusterProvider, cps map[string]provider.CloudProvider, dp provider.SSHKeyProvider, versions map[string]*apiv1.MasterVersion) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, nil
	}
}

func getNodeEndpointV2(kp provider.ClusterProvider, cps map[string]provider.CloudProvider, dp provider.SSHKeyProvider, versions map[string]*apiv1.MasterVersion) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, nil
	}
}

func deleteNodeEndpointV2(kp provider.ClusterProvider, cps map[string]provider.CloudProvider, dp provider.SSHKeyProvider, versions map[string]*apiv1.MasterVersion) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return nil, nil
	}
}
