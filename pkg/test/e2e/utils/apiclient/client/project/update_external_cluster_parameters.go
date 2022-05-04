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

// NewUpdateExternalClusterParams creates a new UpdateExternalClusterParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewUpdateExternalClusterParams() *UpdateExternalClusterParams {
	return &UpdateExternalClusterParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewUpdateExternalClusterParamsWithTimeout creates a new UpdateExternalClusterParams object
// with the ability to set a timeout on a request.
func NewUpdateExternalClusterParamsWithTimeout(timeout time.Duration) *UpdateExternalClusterParams {
	return &UpdateExternalClusterParams{
		timeout: timeout,
	}
}

// NewUpdateExternalClusterParamsWithContext creates a new UpdateExternalClusterParams object
// with the ability to set a context for a request.
func NewUpdateExternalClusterParamsWithContext(ctx context.Context) *UpdateExternalClusterParams {
	return &UpdateExternalClusterParams{
		Context: ctx,
	}
}

// NewUpdateExternalClusterParamsWithHTTPClient creates a new UpdateExternalClusterParams object
// with the ability to set a custom HTTPClient for a request.
func NewUpdateExternalClusterParamsWithHTTPClient(client *http.Client) *UpdateExternalClusterParams {
	return &UpdateExternalClusterParams{
		HTTPClient: client,
	}
}

/* UpdateExternalClusterParams contains all the parameters to send to the API endpoint
   for the update external cluster operation.

   Typically these are written to a http.Request.
*/
type UpdateExternalClusterParams struct {

	// Body.
	Body UpdateExternalClusterBody

	// ClusterID.
	ClusterID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the update external cluster params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpdateExternalClusterParams) WithDefaults() *UpdateExternalClusterParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the update external cluster params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpdateExternalClusterParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the update external cluster params
func (o *UpdateExternalClusterParams) WithTimeout(timeout time.Duration) *UpdateExternalClusterParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the update external cluster params
func (o *UpdateExternalClusterParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the update external cluster params
func (o *UpdateExternalClusterParams) WithContext(ctx context.Context) *UpdateExternalClusterParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the update external cluster params
func (o *UpdateExternalClusterParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the update external cluster params
func (o *UpdateExternalClusterParams) WithHTTPClient(client *http.Client) *UpdateExternalClusterParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the update external cluster params
func (o *UpdateExternalClusterParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the update external cluster params
func (o *UpdateExternalClusterParams) WithBody(body UpdateExternalClusterBody) *UpdateExternalClusterParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the update external cluster params
func (o *UpdateExternalClusterParams) SetBody(body UpdateExternalClusterBody) {
	o.Body = body
}

// WithClusterID adds the clusterID to the update external cluster params
func (o *UpdateExternalClusterParams) WithClusterID(clusterID string) *UpdateExternalClusterParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the update external cluster params
func (o *UpdateExternalClusterParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the update external cluster params
func (o *UpdateExternalClusterParams) WithProjectID(projectID string) *UpdateExternalClusterParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the update external cluster params
func (o *UpdateExternalClusterParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *UpdateExternalClusterParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if err := r.SetBodyParam(o.Body); err != nil {
		return err
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
