// Code generated by go-swagger; DO NOT EDIT.

package gcp

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

// NewListGCPSubnetworksNoCredentialsV2Params creates a new ListGCPSubnetworksNoCredentialsV2Params object
// with the default values initialized.
func NewListGCPSubnetworksNoCredentialsV2Params() *ListGCPSubnetworksNoCredentialsV2Params {
	var ()
	return &ListGCPSubnetworksNoCredentialsV2Params{

		timeout: cr.DefaultTimeout,
	}
}

// NewListGCPSubnetworksNoCredentialsV2ParamsWithTimeout creates a new ListGCPSubnetworksNoCredentialsV2Params object
// with the default values initialized, and the ability to set a timeout on a request
func NewListGCPSubnetworksNoCredentialsV2ParamsWithTimeout(timeout time.Duration) *ListGCPSubnetworksNoCredentialsV2Params {
	var ()
	return &ListGCPSubnetworksNoCredentialsV2Params{

		timeout: timeout,
	}
}

// NewListGCPSubnetworksNoCredentialsV2ParamsWithContext creates a new ListGCPSubnetworksNoCredentialsV2Params object
// with the default values initialized, and the ability to set a context for a request
func NewListGCPSubnetworksNoCredentialsV2ParamsWithContext(ctx context.Context) *ListGCPSubnetworksNoCredentialsV2Params {
	var ()
	return &ListGCPSubnetworksNoCredentialsV2Params{

		Context: ctx,
	}
}

// NewListGCPSubnetworksNoCredentialsV2ParamsWithHTTPClient creates a new ListGCPSubnetworksNoCredentialsV2Params object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListGCPSubnetworksNoCredentialsV2ParamsWithHTTPClient(client *http.Client) *ListGCPSubnetworksNoCredentialsV2Params {
	var ()
	return &ListGCPSubnetworksNoCredentialsV2Params{
		HTTPClient: client,
	}
}

/*ListGCPSubnetworksNoCredentialsV2Params contains all the parameters to send to the API endpoint
for the list g c p subnetworks no credentials v2 operation typically these are written to a http.Request
*/
type ListGCPSubnetworksNoCredentialsV2Params struct {

	/*Network*/
	Network *string
	/*ClusterID*/
	ClusterID string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) WithTimeout(timeout time.Duration) *ListGCPSubnetworksNoCredentialsV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) WithContext(ctx context.Context) *ListGCPSubnetworksNoCredentialsV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) WithHTTPClient(client *http.Client) *ListGCPSubnetworksNoCredentialsV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithNetwork adds the network to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) WithNetwork(network *string) *ListGCPSubnetworksNoCredentialsV2Params {
	o.SetNetwork(network)
	return o
}

// SetNetwork adds the network to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) SetNetwork(network *string) {
	o.Network = network
}

// WithClusterID adds the clusterID to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) WithClusterID(clusterID string) *ListGCPSubnetworksNoCredentialsV2Params {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) WithProjectID(projectID string) *ListGCPSubnetworksNoCredentialsV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list g c p subnetworks no credentials v2 params
func (o *ListGCPSubnetworksNoCredentialsV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListGCPSubnetworksNoCredentialsV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Network != nil {

		// header param Network
		if err := r.SetHeaderParam("Network", *o.Network); err != nil {
			return err
		}

	}

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
		return err
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
