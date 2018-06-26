package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// ClustersReq represent a request for clusters specific data
// swagger:parameters listClusters listClustersV3
type ClustersReq struct {
	DCReq
}

func decodeClustersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClustersReq

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

// ClusterReq represent a request for cluster specific data
// swagger:parameters getCluster deleteCluster getClusterKubeconfig getClusterNodes getClusterV3 getClusterKubeconfigV3 deleteClusterV3 getClusterUpdatesV3
type ClusterReq struct {
	DCReq
	// in: path
	ClusterName string `json:"cluster"`
}

func decodeClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClusterReq
	req.ClusterName = mux.Vars(r)["cluster"]

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

// NodesV2Req represent a request to fetch all cluster nodes
// swagger:parameters getClusterNodesV3 getClusterNodesV2
type NodesV2Req struct {
	ClusterReq
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func decodeNodesV2Req(c context.Context, r *http.Request) (interface{}, error) {
	var req NodesV2Req
	req.ClusterName = mux.Vars(r)["cluster"]
	req.HideInitialConditions, _ = strconv.ParseBool(r.URL.Query().Get("hideInitialConditions"))

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = cr.(ClusterReq)

	return req, nil
}

// CreateClusterReqBody represents the request body for a create cluster request
type CreateClusterReqBody struct {
	Cluster *kubermaticv1.Cluster `json:"cluster"`
}

// UpdateClusterReq represent a update request for a specific cluster
// swagger:parameters updateClusterV3
type UpdateClusterReq struct {
	ClusterReq
	// in: body
	Body CreateClusterReqBody
}

func decodeUpdateClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req UpdateClusterReq
	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = cr.(ClusterReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body.Cluster); err != nil {
		return nil, err
	}

	return req, nil
}

// NewClusterReq represent a request for clusters specific data
// swagger:parameters createCluster createClusterV3
type NewClusterReq struct {
	DCReq
	// in: body
	Body NewClusterReqBody
}

// NewClusterReqBody represents the body of a new cluster request
type NewClusterReqBody struct {
	Cluster *kubermaticv1.ClusterSpec `json:"cluster"`
	SSHKeys []string                  `json:"sshKeys"`
}

func decodeNewClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NewClusterReq

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// DCsReq represent a request for datacenters specific data
type DCsReq struct{}

func decodeDatacentersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DCsReq

	return req, nil
}

//DCGetter defines functionality to retrieve a datacenter name
type DCGetter interface {
	GetDC() string
}

// DCReq represent a request for datacenter specific data
// swagger:parameters getDatacenter
type DCReq struct {
	// in: path
	DC string `json:"dc"`
}

// GetDC returns the name of the datacenter in the request
func (req DCReq) GetDC() string {
	return req.DC
}

func decodeDcReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DCReq

	req.DC = mux.Vars(r)["dc"]
	return req, nil
}

// DoSizesReq represent a request for digitalocean sizes
type DoSizesReq struct {
	DoToken string
}

func decodeDoSizesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DoSizesReq

	req.DoToken = r.Header.Get("DoToken")
	return req, nil
}

// AzureSizeReq represent a request for Azure VM sizes
type AzureSizeReq struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	ClientSecret   string
	Location       string
}

func decodeAzureSizesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AzureSizeReq

	req.SubscriptionID = r.Header.Get("SubscriptionID")
	req.TenantID = r.Header.Get("TenantID")
	req.ClientID = r.Header.Get("ClientID")
	req.ClientSecret = r.Header.Get("ClientSecret")
	req.Location = r.Header.Get("Location")
	return req, nil
}

// OpenstackSizeReq represent a request for openstack sizes
type OpenstackSizeReq struct {
	Username       string
	Password       string
	Tenant         string
	Domain         string
	DatacenterName string
}

func decodeOpenstackSizeReq(c context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackSizeReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.Tenant = r.Header.Get("Tenant")
	req.Domain = r.Header.Get("Domain")
	req.DatacenterName = r.Header.Get("DatacenterName")

	return req, nil
}

// OpenstackTenantReq represent a request for openstack tenants
type OpenstackTenantReq struct {
	Username       string
	Password       string
	Domain         string
	DatacenterName string
}

func decodeOpenstackTenantReq(c context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackTenantReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.Domain = r.Header.Get("Domain")
	req.DatacenterName = r.Header.Get("DatacenterName")

	return req, nil
}

func decodeKubeconfigReq(c context.Context, r *http.Request) (interface{}, error) {
	req, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// CreateNodesReq represent a request for specific data to create a node
// swagger:parameters createClusterNodes
type CreateNodesReq struct {
	ClusterReq
	// in: body
	Body CreateNodesReqBody
}

// CreateNodesReqBody represents the request body of a create nodes request
type CreateNodesReqBody struct {
	Instances int            `json:"instances"`
	Spec      apiv1.NodeSpec `json:"spec"`
}

// NodeReq represent a request for node specific data
// swagger:parameters getClusterNode deleteClusterNode
type NodeReq struct {
	ClusterReq
	// in: path
	NodeName string `json:"node"`
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
}

func decodeNodeReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NodeReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterReq = cr.(ClusterReq)
	req.NodeName = mux.Vars(r)["node"]
	req.HideInitialConditions, _ = strconv.ParseBool(r.URL.Query().Get("hideInitialConditions"))

	return req, nil
}

// CreateSSHKeyReq represent a request for specific data to create a new SSH key
type CreateSSHKeyReq struct {
	apiv1.SSHKey
}

func decodeCreateSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req CreateSSHKeyReq
	req.SSHKey = apiv1.SSHKey{}

	if err := json.NewDecoder(r.Body).Decode(&req.SSHKey); err != nil {
		return nil, errors.NewBadRequest("Error parsing the input, got %q", err.Error())
	}

	return req, nil
}

// DeleteSSHKeyReq represent a request for deleting a SSH key
// swagger:parameters deleteSSHKey
type DeleteSSHKeyReq struct {
	// in: path
	MetaName string `json:"meta_name"`
}

func decodeDeleteSSHKeyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DeleteSSHKeyReq
	var ok bool
	if req.MetaName, ok = mux.Vars(r)["meta_name"]; !ok {
		return nil, fmt.Errorf("delte key needs a parameter 'meta_name'")
	}

	return req, nil
}

// ListSSHKeyReq represent a request for listing all user SSH keys
type ListSSHKeyReq struct{}

func decodeListSSHKeyReq(c context.Context, _ *http.Request) (interface{}, error) {
	var req ListSSHKeyReq
	return req, nil
}

func decodeEmptyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req struct{}
	return req, nil
}
