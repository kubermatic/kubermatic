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

// NewListOpenstackSubnetsNoCredentialsV2Params creates a new ListOpenstackSubnetsNoCredentialsV2Params object
// with the default values initialized.
func NewListOpenstackSubnetsNoCredentialsV2Params() *ListOpenstackSubnetsNoCredentialsV2Params {
	var ()
	return &ListOpenstackSubnetsNoCredentialsV2Params{

		timeout: cr.DefaultTimeout,
	}
}

// NewListOpenstackSubnetsNoCredentialsV2ParamsWithTimeout creates a new ListOpenstackSubnetsNoCredentialsV2Params object
// with the default values initialized, and the ability to set a timeout on a request
func NewListOpenstackSubnetsNoCredentialsV2ParamsWithTimeout(timeout time.Duration) *ListOpenstackSubnetsNoCredentialsV2Params {
	var ()
	return &ListOpenstackSubnetsNoCredentialsV2Params{

		timeout: timeout,
	}
}

// NewListOpenstackSubnetsNoCredentialsV2ParamsWithContext creates a new ListOpenstackSubnetsNoCredentialsV2Params object
// with the default values initialized, and the ability to set a context for a request
func NewListOpenstackSubnetsNoCredentialsV2ParamsWithContext(ctx context.Context) *ListOpenstackSubnetsNoCredentialsV2Params {
	var ()
	return &ListOpenstackSubnetsNoCredentialsV2Params{

		Context: ctx,
	}
}

// NewListOpenstackSubnetsNoCredentialsV2ParamsWithHTTPClient creates a new ListOpenstackSubnetsNoCredentialsV2Params object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListOpenstackSubnetsNoCredentialsV2ParamsWithHTTPClient(client *http.Client) *ListOpenstackSubnetsNoCredentialsV2Params {
	var ()
	return &ListOpenstackSubnetsNoCredentialsV2Params{
		HTTPClient: client,
	}
}

/*ListOpenstackSubnetsNoCredentialsV2Params contains all the parameters to send to the API endpoint
for the list openstack subnets no credentials v2 operation typically these are written to a http.Request
*/
type ListOpenstackSubnetsNoCredentialsV2Params struct {

	/*ClusterID*/
	ClusterID string
	/*NetworkID*/
	NetworkID *string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) WithTimeout(timeout time.Duration) *ListOpenstackSubnetsNoCredentialsV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) WithContext(ctx context.Context) *ListOpenstackSubnetsNoCredentialsV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) WithHTTPClient(client *http.Client) *ListOpenstackSubnetsNoCredentialsV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) WithClusterID(clusterID string) *ListOpenstackSubnetsNoCredentialsV2Params {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithNetworkID adds the networkID to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) WithNetworkID(networkID *string) *ListOpenstackSubnetsNoCredentialsV2Params {
	o.SetNetworkID(networkID)
	return o
}

// SetNetworkID adds the networkId to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) SetNetworkID(networkID *string) {
	o.NetworkID = networkID
}

// WithProjectID adds the projectID to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) WithProjectID(projectID string) *ListOpenstackSubnetsNoCredentialsV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list openstack subnets no credentials v2 params
func (o *ListOpenstackSubnetsNoCredentialsV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListOpenstackSubnetsNoCredentialsV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
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
