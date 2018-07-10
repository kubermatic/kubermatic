package handler

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	"fmt"

	"github.com/gorilla/mux"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	apierrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func getNodeForCluster(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewNodeReq)
		user := ctx.Value(userCRContextKey).(*kubermaticapiv1.User)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)

		project, err := projectProvider.Get(user, req.ProjectName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(user, project, req.ClusterName)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		machineClient, err := clusterProvider.GetMachineClientForCustomerCluster(cluster)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		kubeClient, err := clusterProvider.GetKubernetesClientForCustomerCluster(cluster)
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		machine, node, err := tryToFindMachineAndNode(req.NodeName, machineClient, kubeClient)
		if err != nil {
			return nil, err
		}
		if machine == nil && node == nil {
			return nil, apierrors.NewNotFound("Node", req.NodeName)
		}

		if machine == nil {
			return convertNodeV2ToNodeV1(outputNode(node, req.HideInitialConditions)), nil
		}

		nodeV2, err := outputMachine(machine, node, req.HideInitialConditions)
		if err != nil {
			return nil, err
		}
		return convertNodeV2ToNodeV1(nodeV2), nil
	}
}

func convertNodeV2ToNodeV1(nodeV2 *apiv2.Node) *apiv1.Node {
	return &apiv1.Node{
		NewObjectMeta: apiv1.NewObjectMeta{
			ID:                nodeV2.Metadata.Name,
			Name:              nodeV2.Metadata.DisplayName,
			CreationTimestamp: nodeV2.Metadata.CreationTimestamp,
			DeletionTimestamp: nodeV2.Metadata.DeletionTimestamp,
		},
		Spec:   nodeV2.Spec,
		Status: nodeV2.Status,
	}
}

// NewNodeReq defines HTTP request for newGetNodeForCluster
// swagger:parameters newGetNodeForCluster
type NewNodeReq struct {
	NewGetClusterReq
	// in: path
	NodeName string `json:"node_name"`
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func decodeGetNodeForCluster(c context.Context, r *http.Request) (interface{}, error) {
	var req NewNodeReq
	clusterName, projectName, err := decodeClusterNameAndProject(c, r)
	if err != nil {
		return nil, err
	}

	nodeName, ok := mux.Vars(r)["node_name"]
	if !ok {
		return nil, fmt.Errorf("'node_name' parameter is required but was not provided")
	}

	req.ProjectName = projectName
	req.ClusterName = clusterName
	req.NodeName = nodeName

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}
