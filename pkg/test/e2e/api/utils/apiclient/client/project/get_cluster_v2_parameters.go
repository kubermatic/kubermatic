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

// NewGetClusterV2Params creates a new GetClusterV2Params object
// with the default values initialized.
func NewGetClusterV2Params() *GetClusterV2Params {
	var ()
	return &GetClusterV2Params{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetClusterV2ParamsWithTimeout creates a new GetClusterV2Params object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetClusterV2ParamsWithTimeout(timeout time.Duration) *GetClusterV2Params {
	var ()
	return &GetClusterV2Params{

		timeout: timeout,
	}
}

// NewGetClusterV2ParamsWithContext creates a new GetClusterV2Params object
// with the default values initialized, and the ability to set a context for a request
func NewGetClusterV2ParamsWithContext(ctx context.Context) *GetClusterV2Params {
	var ()
	return &GetClusterV2Params{

		Context: ctx,
	}
}

// NewGetClusterV2ParamsWithHTTPClient creates a new GetClusterV2Params object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetClusterV2ParamsWithHTTPClient(client *http.Client) *GetClusterV2Params {
	var ()
	return &GetClusterV2Params{
		HTTPClient: client,
	}
}

/*GetClusterV2Params contains all the parameters to send to the API endpoint
for the get cluster v2 operation typically these are written to a http.Request
*/
type GetClusterV2Params struct {

	/*ClusterID*/
	ClusterID string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get cluster v2 params
func (o *GetClusterV2Params) WithTimeout(timeout time.Duration) *GetClusterV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get cluster v2 params
func (o *GetClusterV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get cluster v2 params
func (o *GetClusterV2Params) WithContext(ctx context.Context) *GetClusterV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get cluster v2 params
func (o *GetClusterV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get cluster v2 params
func (o *GetClusterV2Params) WithHTTPClient(client *http.Client) *GetClusterV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get cluster v2 params
func (o *GetClusterV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the get cluster v2 params
func (o *GetClusterV2Params) WithClusterID(clusterID string) *GetClusterV2Params {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get cluster v2 params
func (o *GetClusterV2Params) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the get cluster v2 params
func (o *GetClusterV2Params) WithProjectID(projectID string) *GetClusterV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get cluster v2 params
func (o *GetClusterV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *GetClusterV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
