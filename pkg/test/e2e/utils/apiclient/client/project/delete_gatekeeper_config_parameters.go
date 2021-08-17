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

// NewDeleteGatekeeperConfigParams creates a new DeleteGatekeeperConfigParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteGatekeeperConfigParams() *DeleteGatekeeperConfigParams {
	return &DeleteGatekeeperConfigParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteGatekeeperConfigParamsWithTimeout creates a new DeleteGatekeeperConfigParams object
// with the ability to set a timeout on a request.
func NewDeleteGatekeeperConfigParamsWithTimeout(timeout time.Duration) *DeleteGatekeeperConfigParams {
	return &DeleteGatekeeperConfigParams{
		timeout: timeout,
	}
}

// NewDeleteGatekeeperConfigParamsWithContext creates a new DeleteGatekeeperConfigParams object
// with the ability to set a context for a request.
func NewDeleteGatekeeperConfigParamsWithContext(ctx context.Context) *DeleteGatekeeperConfigParams {
	return &DeleteGatekeeperConfigParams{
		Context: ctx,
	}
}

// NewDeleteGatekeeperConfigParamsWithHTTPClient creates a new DeleteGatekeeperConfigParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteGatekeeperConfigParamsWithHTTPClient(client *http.Client) *DeleteGatekeeperConfigParams {
	return &DeleteGatekeeperConfigParams{
		HTTPClient: client,
	}
}

/* DeleteGatekeeperConfigParams contains all the parameters to send to the API endpoint
   for the delete gatekeeper config operation.

   Typically these are written to a http.Request.
*/
type DeleteGatekeeperConfigParams struct {

	// ClusterID.
	ClusterID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete gatekeeper config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteGatekeeperConfigParams) WithDefaults() *DeleteGatekeeperConfigParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete gatekeeper config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteGatekeeperConfigParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete gatekeeper config params
func (o *DeleteGatekeeperConfigParams) WithTimeout(timeout time.Duration) *DeleteGatekeeperConfigParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete gatekeeper config params
func (o *DeleteGatekeeperConfigParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete gatekeeper config params
func (o *DeleteGatekeeperConfigParams) WithContext(ctx context.Context) *DeleteGatekeeperConfigParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete gatekeeper config params
func (o *DeleteGatekeeperConfigParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete gatekeeper config params
func (o *DeleteGatekeeperConfigParams) WithHTTPClient(client *http.Client) *DeleteGatekeeperConfigParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete gatekeeper config params
func (o *DeleteGatekeeperConfigParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the delete gatekeeper config params
func (o *DeleteGatekeeperConfigParams) WithClusterID(clusterID string) *DeleteGatekeeperConfigParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the delete gatekeeper config params
func (o *DeleteGatekeeperConfigParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the delete gatekeeper config params
func (o *DeleteGatekeeperConfigParams) WithProjectID(projectID string) *DeleteGatekeeperConfigParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the delete gatekeeper config params
func (o *DeleteGatekeeperConfigParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteGatekeeperConfigParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
