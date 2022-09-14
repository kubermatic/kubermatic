// Code generated by go-swagger; DO NOT EDIT.

package addon

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

// NewListAddonsParams creates a new ListAddonsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListAddonsParams() *ListAddonsParams {
	return &ListAddonsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListAddonsParamsWithTimeout creates a new ListAddonsParams object
// with the ability to set a timeout on a request.
func NewListAddonsParamsWithTimeout(timeout time.Duration) *ListAddonsParams {
	return &ListAddonsParams{
		timeout: timeout,
	}
}

// NewListAddonsParamsWithContext creates a new ListAddonsParams object
// with the ability to set a context for a request.
func NewListAddonsParamsWithContext(ctx context.Context) *ListAddonsParams {
	return &ListAddonsParams{
		Context: ctx,
	}
}

// NewListAddonsParamsWithHTTPClient creates a new ListAddonsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListAddonsParamsWithHTTPClient(client *http.Client) *ListAddonsParams {
	return &ListAddonsParams{
		HTTPClient: client,
	}
}

/*
ListAddonsParams contains all the parameters to send to the API endpoint

	for the list addons operation.

	Typically these are written to a http.Request.
*/
type ListAddonsParams struct {

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

// WithDefaults hydrates default values in the list addons params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAddonsParams) WithDefaults() *ListAddonsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list addons params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAddonsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list addons params
func (o *ListAddonsParams) WithTimeout(timeout time.Duration) *ListAddonsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list addons params
func (o *ListAddonsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list addons params
func (o *ListAddonsParams) WithContext(ctx context.Context) *ListAddonsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list addons params
func (o *ListAddonsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list addons params
func (o *ListAddonsParams) WithHTTPClient(client *http.Client) *ListAddonsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list addons params
func (o *ListAddonsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list addons params
func (o *ListAddonsParams) WithClusterID(clusterID string) *ListAddonsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list addons params
func (o *ListAddonsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the list addons params
func (o *ListAddonsParams) WithDC(dc string) *ListAddonsParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list addons params
func (o *ListAddonsParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the list addons params
func (o *ListAddonsParams) WithProjectID(projectID string) *ListAddonsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list addons params
func (o *ListAddonsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListAddonsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
