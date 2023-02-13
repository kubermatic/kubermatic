// Code generated by go-swagger; DO NOT EDIT.

package anexia

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

// NewListAnexiaVlansNoCredentialsV2Params creates a new ListAnexiaVlansNoCredentialsV2Params object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListAnexiaVlansNoCredentialsV2Params() *ListAnexiaVlansNoCredentialsV2Params {
	return &ListAnexiaVlansNoCredentialsV2Params{
		timeout: cr.DefaultTimeout,
	}
}

// NewListAnexiaVlansNoCredentialsV2ParamsWithTimeout creates a new ListAnexiaVlansNoCredentialsV2Params object
// with the ability to set a timeout on a request.
func NewListAnexiaVlansNoCredentialsV2ParamsWithTimeout(timeout time.Duration) *ListAnexiaVlansNoCredentialsV2Params {
	return &ListAnexiaVlansNoCredentialsV2Params{
		timeout: timeout,
	}
}

// NewListAnexiaVlansNoCredentialsV2ParamsWithContext creates a new ListAnexiaVlansNoCredentialsV2Params object
// with the ability to set a context for a request.
func NewListAnexiaVlansNoCredentialsV2ParamsWithContext(ctx context.Context) *ListAnexiaVlansNoCredentialsV2Params {
	return &ListAnexiaVlansNoCredentialsV2Params{
		Context: ctx,
	}
}

// NewListAnexiaVlansNoCredentialsV2ParamsWithHTTPClient creates a new ListAnexiaVlansNoCredentialsV2Params object
// with the ability to set a custom HTTPClient for a request.
func NewListAnexiaVlansNoCredentialsV2ParamsWithHTTPClient(client *http.Client) *ListAnexiaVlansNoCredentialsV2Params {
	return &ListAnexiaVlansNoCredentialsV2Params{
		HTTPClient: client,
	}
}

/* ListAnexiaVlansNoCredentialsV2Params contains all the parameters to send to the API endpoint
   for the list anexia vlans no credentials v2 operation.

   Typically these are written to a http.Request.
*/
type ListAnexiaVlansNoCredentialsV2Params struct {

	// ClusterID.
	ClusterID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list anexia vlans no credentials v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAnexiaVlansNoCredentialsV2Params) WithDefaults() *ListAnexiaVlansNoCredentialsV2Params {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list anexia vlans no credentials v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAnexiaVlansNoCredentialsV2Params) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list anexia vlans no credentials v2 params
func (o *ListAnexiaVlansNoCredentialsV2Params) WithTimeout(timeout time.Duration) *ListAnexiaVlansNoCredentialsV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list anexia vlans no credentials v2 params
func (o *ListAnexiaVlansNoCredentialsV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list anexia vlans no credentials v2 params
func (o *ListAnexiaVlansNoCredentialsV2Params) WithContext(ctx context.Context) *ListAnexiaVlansNoCredentialsV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list anexia vlans no credentials v2 params
func (o *ListAnexiaVlansNoCredentialsV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list anexia vlans no credentials v2 params
func (o *ListAnexiaVlansNoCredentialsV2Params) WithHTTPClient(client *http.Client) *ListAnexiaVlansNoCredentialsV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list anexia vlans no credentials v2 params
func (o *ListAnexiaVlansNoCredentialsV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list anexia vlans no credentials v2 params
func (o *ListAnexiaVlansNoCredentialsV2Params) WithClusterID(clusterID string) *ListAnexiaVlansNoCredentialsV2Params {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list anexia vlans no credentials v2 params
func (o *ListAnexiaVlansNoCredentialsV2Params) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the list anexia vlans no credentials v2 params
func (o *ListAnexiaVlansNoCredentialsV2Params) WithProjectID(projectID string) *ListAnexiaVlansNoCredentialsV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list anexia vlans no credentials v2 params
func (o *ListAnexiaVlansNoCredentialsV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListAnexiaVlansNoCredentialsV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

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