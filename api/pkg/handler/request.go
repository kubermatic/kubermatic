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

// ListClustersReq represent a request for clusters specific data
// swagger:parameters listClusters listClustersV3
type ListClustersReq struct {
	DCReq
}

func decodeClustersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ListClustersReq

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

// GetClusterReq represent a request for cluster specific data
// swagger:parameters getClusterV3 getClusterKubeconfigV3 deleteClusterV3 getClusterUpdatesV3 createNodesHandlerV3 getPossibleClusterUpgradesV3
type GetClusterReq struct {
	DCReq
	// in: path
	ClusterName string `json:"cluster"`
}

func decodeClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GetClusterReq
	req.ClusterName = mux.Vars(r)["cluster"]

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

	return req, nil
}

// CreateClusterReqBody represents the request body for a create cluster request
type CreateClusterReqBody struct {
	Cluster *kubermaticv1.Cluster `json:"cluster"`
}

// UpdateClusterReq represent a update request for a specific cluster
// swagger:parameters updateClusterV3
type UpdateClusterReq struct {
	GetClusterReq
	// in: body
	Body CreateClusterReqBody
}

func decodeUpdateClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req UpdateClusterReq
	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(GetClusterReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body.Cluster); err != nil {
		return nil, err
	}

	return req, nil
}

// ClusterReq represent a request for clusters specific data
// swagger:parameters createCluster createClusterV3
type ClusterReq struct {
	DCReq
	// in: body
	Body ClusterReqBody
}

// ClusterReqBody represents the body of a new cluster request
type ClusterReqBody struct {
	Cluster *kubermaticv1.ClusterSpec `json:"cluster"`
	SSHKeys []string                  `json:"sshKeys"`
}

func decodeNewClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req ClusterReq

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

func decodeProjectPathReq(c context.Context, r *http.Request) (string, error) {
	// project_id is actually an internal name of the object
	projectName := mux.Vars(r)["project_id"]
	if projectName == "" {
		return "", fmt.Errorf("'project_id' parameter is required but was not provided")
	}
	return projectName, nil
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

// OpenstackReq represent a request for openstack
type OpenstackReq struct {
	Username       string
	Password       string
	Domain         string
	Tenant         string
	DatacenterName string
}

func decodeOpenstackReq(c context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.Tenant = r.Header.Get("Tenant")
	req.Domain = r.Header.Get("Domain")
	req.DatacenterName = r.Header.Get("DatacenterName")

	return req, nil
}

// OpenstackSubnetReq represent a request for openstack subnets
// swagger:parameters listOpenstackSubnets
type OpenstackSubnetReq struct {
	OpenstackReq
	// in: query
	NetworkID string
}

func decodeOpenstackSubnetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackSubnetReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.Domain = r.Header.Get("Domain")
	req.Tenant = r.Header.Get("Tenant")
	req.DatacenterName = r.Header.Get("DatacenterName")
	req.NetworkID = r.URL.Query().Get("network_id")
	if req.NetworkID == "" {
		return nil, fmt.Errorf("get openstack subnets needs a parameter 'network_id'")
	}
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

// NodeReq represent a request for node specific data
// swagger:parameters deleteNodeHandlerV3 getNodeHandlerV3
type NodeReq struct {
	GetClusterReq
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
	req.GetClusterReq = cr.(GetClusterReq)
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

// ListSSHKeyReq represent a request for listing all user SSH Keys
type ListSSHKeyReq struct{}

func decodeListSSHKeyReq(c context.Context, _ *http.Request) (interface{}, error) {
	var req ListSSHKeyReq
	return req, nil
}

func decodeEmptyReq(c context.Context, r *http.Request) (interface{}, error) {
	var req struct{}
	return req, nil
}

// VSphereNetworksReq represent a request for vsphere networks
type VSphereNetworksReq struct {
	Username       string
	Password       string
	DatacenterName string
}

func decodeVSphereNetworksReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereNetworksReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.DatacenterName = r.Header.Get("DatacenterName")

	return req, nil
}
