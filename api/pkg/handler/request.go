package handler

import (
	"context"
	"fmt"
	"net/http"

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

func decodeProjectRequest(c context.Context, r *http.Request) (interface{}, error) {
	return ProjectReq{
		ProjectID: mux.Vars(r)["project_id"],
	}, nil
}

// ListClustersReq represent a request for clusters specific data
// swagger:parameters listClusters listClustersV3
type ListClustersReq struct {
	LegacyDCReq
}

// LegacyGetClusterReq represent a request for cluster specific data
// swagger:parameters getClusterV3 getClusterKubeconfigV3 deleteClusterV3 getClusterUpdatesV3 createNodesHandlerV3 legacyGetPossibleClusterUpgradesV3
type LegacyGetClusterReq struct {
	LegacyDCReq
	// in: path
	ClusterName string `json:"cluster"`
}

// GetClusterReq represent a request for cluster specific data
// swagger:parameters getClusterUpgrades clusterMetricsHandler
type GetClusterReq struct {
	DCReq
	// in: path
	ClusterID string `json:"cluster_id"`
}

func decodeClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GetClusterReq

	clusterID, err := decodeClusterID(c, r)
	if err != nil {
		return nil, err
	}
	req.ClusterID = clusterID

	dcr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(DCReq)

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

// LegacyDCReq represent a request for datacenter specific data
// swagger:parameters getDatacenter
type LegacyDCReq struct {
	// in: path
	DC string `json:"dc"`
}

// GetDC returns the name of the datacenter in the request
func (req LegacyDCReq) GetDC() string {
	return req.DC
}

func decodeLegacyDcReq(c context.Context, r *http.Request) (interface{}, error) {
	var req LegacyDCReq

	req.DC = mux.Vars(r)["dc"]
	return req, nil
}

func decodeDcReq(c context.Context, r *http.Request) (interface{}, error) {
	projectReq, err := decodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	return DCReq{
		DC:         mux.Vars(r)["dc"],
		ProjectReq: projectReq.(ProjectReq),
	}, nil
}

// DoSizesNoCredentialsReq represent a request for digitalocean sizes EP,
// note that the request doesn't have credentials for autN
// swagger:parameters listDigitaloceanSizesNoCredentials
type DoSizesNoCredentialsReq struct {
	NewGetClusterReq
}

func decodeDoSizesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DoSizesNoCredentialsReq
	cr, err := newDecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.NewGetClusterReq = cr.(NewGetClusterReq)
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

// AzureSizeNoCredentialsReq represent a request for Azure VM sizes
// note that the request doesn't have credentials for authN
// swagger:parameters listAzureSizesNoCredentials
type AzureSizeNoCredentialsReq struct {
	NewGetClusterReq
}

func decodeAzureSizesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AzureSizeNoCredentialsReq
	cr, err := newDecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.NewGetClusterReq = cr.(NewGetClusterReq)
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

// OpenstackNoCredentialsReq represent a request for openstack
// swagger:parameters listOpenstackSizesNoCredentials listOpenstackTenantsNoCredentials listOpenstackNetworksNoCredentials listOpenstackSecurityGroupsNoCredentials
type OpenstackNoCredentialsReq struct {
	NewGetClusterReq
}

func decodeOpenstackNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackNoCredentialsReq
	cr, err := newDecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.NewGetClusterReq = cr.(NewGetClusterReq)
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

// OpenstackSubnetNoCredentialsReq represent a request for openstack subnets
// swagger:parameters listOpenstackSubnetsNoCredentials
type OpenstackSubnetNoCredentialsReq struct {
	OpenstackNoCredentialsReq
	// in: query
	NetworkID string
}

func decodeOpenstackSubnetNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackSubnetNoCredentialsReq
	lr, err := decodeOpenstackNoCredentialsReq(c, r)
	if err != nil {
		return nil, err
	}
	req.OpenstackNoCredentialsReq = lr.(OpenstackNoCredentialsReq)

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

// NodeReq represent a request for node specific data
// swagger:parameters deleteNodeHandlerV3 getNodeHandlerV3
type NodeReq struct {
	LegacyGetClusterReq
	// in: path
	NodeName string `json:"node"`
	// in: query
	HideInitialConditions bool `json:"hideInitialConditions"`
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

// VSphereNetworksNoCredentialsReq represent a request for vsphere networks
// swagger:parameters listVSphereNetworksNoCredentials
type VSphereNetworksNoCredentialsReq struct {
	NewGetClusterReq
}

func decodeVSphereNetworksNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereNetworksNoCredentialsReq
	lr, err := newDecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.NewGetClusterReq = lr.(NewGetClusterReq)
	return req, nil
}
