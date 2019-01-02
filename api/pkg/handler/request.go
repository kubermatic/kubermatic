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
	GetClusterReq
}

func decodeDoSizesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DoSizesNoCredentialsReq
	cr, err := decodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(GetClusterReq)
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
	GetClusterReq
}

func decodeAzureSizesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AzureSizeNoCredentialsReq
	cr, err := decodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(GetClusterReq)
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
	GetClusterReq
}

func decodeOpenstackNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req OpenstackNoCredentialsReq
	cr, err := decodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(GetClusterReq)
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
	GetClusterReq
}

func decodeVSphereNetworksNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereNetworksNoCredentialsReq
	lr, err := decodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = lr.(GetClusterReq)
	return req, nil
}
