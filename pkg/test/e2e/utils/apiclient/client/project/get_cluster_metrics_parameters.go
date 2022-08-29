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

// NewGetClusterMetricsParams creates a new GetClusterMetricsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetClusterMetricsParams() *GetClusterMetricsParams {
	return &GetClusterMetricsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetClusterMetricsParamsWithTimeout creates a new GetClusterMetricsParams object
// with the ability to set a timeout on a request.
func NewGetClusterMetricsParamsWithTimeout(timeout time.Duration) *GetClusterMetricsParams {
	return &GetClusterMetricsParams{
		timeout: timeout,
	}
}

// NewGetClusterMetricsParamsWithContext creates a new GetClusterMetricsParams object
// with the ability to set a context for a request.
func NewGetClusterMetricsParamsWithContext(ctx context.Context) *GetClusterMetricsParams {
	return &GetClusterMetricsParams{
		Context: ctx,
	}
}

// NewGetClusterMetricsParamsWithHTTPClient creates a new GetClusterMetricsParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetClusterMetricsParamsWithHTTPClient(client *http.Client) *GetClusterMetricsParams {
	return &GetClusterMetricsParams{
		HTTPClient: client,
	}
}

/*
GetClusterMetricsParams contains all the parameters to send to the API endpoint

	for the get cluster metrics operation.

	Typically these are written to a http.Request.
*/
type GetClusterMetricsParams struct {

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

// WithDefaults hydrates default values in the get cluster metrics params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetClusterMetricsParams) WithDefaults() *GetClusterMetricsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get cluster metrics params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetClusterMetricsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get cluster metrics params
func (o *GetClusterMetricsParams) WithTimeout(timeout time.Duration) *GetClusterMetricsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get cluster metrics params
func (o *GetClusterMetricsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get cluster metrics params
func (o *GetClusterMetricsParams) WithContext(ctx context.Context) *GetClusterMetricsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get cluster metrics params
func (o *GetClusterMetricsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get cluster metrics params
func (o *GetClusterMetricsParams) WithHTTPClient(client *http.Client) *GetClusterMetricsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get cluster metrics params
func (o *GetClusterMetricsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the get cluster metrics params
func (o *GetClusterMetricsParams) WithClusterID(clusterID string) *GetClusterMetricsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get cluster metrics params
func (o *GetClusterMetricsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the get cluster metrics params
func (o *GetClusterMetricsParams) WithDC(dc string) *GetClusterMetricsParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the get cluster metrics params
func (o *GetClusterMetricsParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the get cluster metrics params
func (o *GetClusterMetricsParams) WithProjectID(projectID string) *GetClusterMetricsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get cluster metrics params
func (o *GetClusterMetricsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *GetClusterMetricsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
