// Code generated by go-swagger; DO NOT EDIT.

package openstack

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewListOpenstackSubnetsNoCredentialsParams creates a new ListOpenstackSubnetsNoCredentialsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListOpenstackSubnetsNoCredentialsParams() *ListOpenstackSubnetsNoCredentialsParams {
	return &ListOpenstackSubnetsNoCredentialsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListOpenstackSubnetsNoCredentialsParamsWithTimeout creates a new ListOpenstackSubnetsNoCredentialsParams object
// with the ability to set a timeout on a request.
func NewListOpenstackSubnetsNoCredentialsParamsWithTimeout(timeout time.Duration) *ListOpenstackSubnetsNoCredentialsParams {
	return &ListOpenstackSubnetsNoCredentialsParams{
		timeout: timeout,
	}
}

// NewListOpenstackSubnetsNoCredentialsParamsWithContext creates a new ListOpenstackSubnetsNoCredentialsParams object
// with the ability to set a context for a request.
func NewListOpenstackSubnetsNoCredentialsParamsWithContext(ctx context.Context) *ListOpenstackSubnetsNoCredentialsParams {
	return &ListOpenstackSubnetsNoCredentialsParams{
		Context: ctx,
	}
}

// NewListOpenstackSubnetsNoCredentialsParamsWithHTTPClient creates a new ListOpenstackSubnetsNoCredentialsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListOpenstackSubnetsNoCredentialsParamsWithHTTPClient(client *http.Client) *ListOpenstackSubnetsNoCredentialsParams {
	return &ListOpenstackSubnetsNoCredentialsParams{
		HTTPClient: client,
	}
}

/*
ListOpenstackSubnetsNoCredentialsParams contains all the parameters to send to the API endpoint

	for the list openstack subnets no credentials operation.

	Typically these are written to a http.Request.
*/
type ListOpenstackSubnetsNoCredentialsParams struct {

	// ClusterID.
	ClusterID string

	// Dc.
	DC string

	// NetworkID.
	NetworkID *string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list openstack subnets no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListOpenstackSubnetsNoCredentialsParams) WithDefaults() *ListOpenstackSubnetsNoCredentialsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list openstack subnets no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListOpenstackSubnetsNoCredentialsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) WithTimeout(timeout time.Duration) *ListOpenstackSubnetsNoCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) WithContext(ctx context.Context) *ListOpenstackSubnetsNoCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) WithHTTPClient(client *http.Client) *ListOpenstackSubnetsNoCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) WithClusterID(clusterID string) *ListOpenstackSubnetsNoCredentialsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) WithDC(dc string) *ListOpenstackSubnetsNoCredentialsParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) SetDC(dc string) {
	o.DC = dc
}

// WithNetworkID adds the networkID to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) WithNetworkID(networkID *string) *ListOpenstackSubnetsNoCredentialsParams {
	o.SetNetworkID(networkID)
	return o
}

// SetNetworkID adds the networkId to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) SetNetworkID(networkID *string) {
	o.NetworkID = networkID
}

// WithProjectID adds the projectID to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) WithProjectID(projectID string) *ListOpenstackSubnetsNoCredentialsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list openstack subnets no credentials params
func (o *ListOpenstackSubnetsNoCredentialsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListOpenstackSubnetsNoCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
		return err
	}

	// path param dc
	if err := r.SetPathParam("dc", o.DC); err != nil {
		return err
	}

	if o.NetworkID != nil {

		// query param network_id
		var qrNetworkID string

		if o.NetworkID != nil {
			qrNetworkID = *o.NetworkID
		}
		qNetworkID := qrNetworkID
		if qNetworkID != "" {

			if err := r.SetQueryParam("network_id", qNetworkID); err != nil {
				return err
			}
		}
	}

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
