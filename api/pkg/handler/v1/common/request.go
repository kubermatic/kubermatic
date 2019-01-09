package common

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// ProjectReq represents a request for project-specific data
type ProjectReq struct {
	// in: path
	ProjectID string `json:"project_id"`
}

// GetProjectID returns the ID of a requested project
func (pr ProjectReq) GetProjectID() string {
	return pr.ProjectID
}

func DecodeProjectRequest(c context.Context, r *http.Request) (interface{}, error) {
	return ProjectReq{
		ProjectID: mux.Vars(r)["project_id"],
	}, nil
}

// ProjectIDGetter knows how to get project ID from the request
type ProjectIDGetter interface {
	GetProjectID() string
}

// DCReq represent a request for datacenter specific data in a given project
type DCReq struct {
	ProjectReq
	// in: path
	DC string `json:"dc"`
}

// GetDC returns the name of the datacenter in the request
func (req DCReq) GetDC() string {
	return req.DC
}

func DecodeDcReq(c context.Context, r *http.Request) (interface{}, error) {
	projectReq, err := DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	return DCReq{
		DC:         mux.Vars(r)["dc"],
		ProjectReq: projectReq.(ProjectReq),
	}, nil
}

// GetClusterReq defines HTTP request for deleteCluster and getClusterKubeconfig endpoints
// swagger:parameters getCluster deleteCluster getClusterKubeconfig getClusterHealth getClusterUpgrades getClusterMetrics
type GetClusterReq struct {
	DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

type DeleteClusterReq struct {
	DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
	// DeleteVolumes if true all cluster PV's and PVC's will be deleted from cluster
	DeleteVolumes bool
	// DeleteVolumes if true all load balancers will be deleted from cluster
	DeleteLoadBalancers bool
}

func DecodeDeleteClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteClusterReq

	clusterReqRaw, err := DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	clusterReq := clusterReqRaw.(GetClusterReq)
	req.DCReq = clusterReq.DCReq
	req.ClusterID = clusterReq.ClusterID

	headerValue := r.Header.Get("DeleteVolumes")
	if len(headerValue) > 0 {
		deleteVolumes, err := strconv.ParseBool(headerValue)
		if err != nil {
			return nil, err
		}
		req.DeleteVolumes = deleteVolumes
	}

	headerValue = r.Header.Get("DeleteLoadBalancers")
	if len(headerValue) > 0 {
		deleteLB, err := strconv.ParseBool(headerValue)
		if err != nil {
			return nil, err
		}
		req.DeleteLoadBalancers = deleteLB
	}

	return req, nil
}

func DecodeGetClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GetClusterReq
	clusterID, err := DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

func DecodeClusterID(c context.Context, r *http.Request) (string, error) {
	clusterID := mux.Vars(r)["cluster_id"]
	if clusterID == "" {
		return "", fmt.Errorf("'cluster_id' parameter is required but was not provided")
	}

	return clusterID, nil
}
