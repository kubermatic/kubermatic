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

// NewGetClusterOidcParams creates a new GetClusterOidcParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetClusterOidcParams() *GetClusterOidcParams {
	return &GetClusterOidcParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetClusterOidcParamsWithTimeout creates a new GetClusterOidcParams object
// with the ability to set a timeout on a request.
func NewGetClusterOidcParamsWithTimeout(timeout time.Duration) *GetClusterOidcParams {
	return &GetClusterOidcParams{
		timeout: timeout,
	}
}

// NewGetClusterOidcParamsWithContext creates a new GetClusterOidcParams object
// with the ability to set a context for a request.
func NewGetClusterOidcParamsWithContext(ctx context.Context) *GetClusterOidcParams {
	return &GetClusterOidcParams{
		Context: ctx,
	}
}

// NewGetClusterOidcParamsWithHTTPClient creates a new GetClusterOidcParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetClusterOidcParamsWithHTTPClient(client *http.Client) *GetClusterOidcParams {
	return &GetClusterOidcParams{
		HTTPClient: client,
	}
}

/*
GetClusterOidcParams contains all the parameters to send to the API endpoint

	for the get cluster oidc operation.

	Typically these are written to a http.Request.
*/
type GetClusterOidcParams struct {

	// ClusterID.
	ClusterID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get cluster oidc params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetClusterOidcParams) WithDefaults() *GetClusterOidcParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get cluster oidc params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetClusterOidcParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get cluster oidc params
func (o *GetClusterOidcParams) WithTimeout(timeout time.Duration) *GetClusterOidcParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get cluster oidc params
func (o *GetClusterOidcParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get cluster oidc params
func (o *GetClusterOidcParams) WithContext(ctx context.Context) *GetClusterOidcParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get cluster oidc params
func (o *GetClusterOidcParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get cluster oidc params
func (o *GetClusterOidcParams) WithHTTPClient(client *http.Client) *GetClusterOidcParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get cluster oidc params
func (o *GetClusterOidcParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the get cluster oidc params
func (o *GetClusterOidcParams) WithClusterID(clusterID string) *GetClusterOidcParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get cluster oidc params
func (o *GetClusterOidcParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the get cluster oidc params
func (o *GetClusterOidcParams) WithProjectID(projectID string) *GetClusterOidcParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get cluster oidc params
func (o *GetClusterOidcParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *GetClusterOidcParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
