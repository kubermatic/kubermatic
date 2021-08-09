// Code generated by go-swagger; DO NOT EDIT.

package project

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

// NewListNamespaceParams creates a new ListNamespaceParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListNamespaceParams() *ListNamespaceParams {
	return &ListNamespaceParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListNamespaceParamsWithTimeout creates a new ListNamespaceParams object
// with the ability to set a timeout on a request.
func NewListNamespaceParamsWithTimeout(timeout time.Duration) *ListNamespaceParams {
	return &ListNamespaceParams{
		timeout: timeout,
	}
}

// NewListNamespaceParamsWithContext creates a new ListNamespaceParams object
// with the ability to set a context for a request.
func NewListNamespaceParamsWithContext(ctx context.Context) *ListNamespaceParams {
	return &ListNamespaceParams{
		Context: ctx,
	}
}

// NewListNamespaceParamsWithHTTPClient creates a new ListNamespaceParams object
// with the ability to set a custom HTTPClient for a request.
func NewListNamespaceParamsWithHTTPClient(client *http.Client) *ListNamespaceParams {
	return &ListNamespaceParams{
		HTTPClient: client,
	}
}

/* ListNamespaceParams contains all the parameters to send to the API endpoint
   for the list namespace operation.

   Typically these are written to a http.Request.
*/
type ListNamespaceParams struct {

	// ClusterID.
	ClusterID string

	// Dc.
	DC string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list namespace params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListNamespaceParams) WithDefaults() *ListNamespaceParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list namespace params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListNamespaceParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list namespace params
func (o *ListNamespaceParams) WithTimeout(timeout time.Duration) *ListNamespaceParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list namespace params
func (o *ListNamespaceParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list namespace params
func (o *ListNamespaceParams) WithContext(ctx context.Context) *ListNamespaceParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list namespace params
func (o *ListNamespaceParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list namespace params
func (o *ListNamespaceParams) WithHTTPClient(client *http.Client) *ListNamespaceParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list namespace params
func (o *ListNamespaceParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list namespace params
func (o *ListNamespaceParams) WithClusterID(clusterID string) *ListNamespaceParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list namespace params
func (o *ListNamespaceParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the list namespace params
func (o *ListNamespaceParams) WithDC(dc string) *ListNamespaceParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list namespace params
func (o *ListNamespaceParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the list namespace params
func (o *ListNamespaceParams) WithProjectID(projectID string) *ListNamespaceParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list namespace params
func (o *ListNamespaceParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListNamespaceParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
