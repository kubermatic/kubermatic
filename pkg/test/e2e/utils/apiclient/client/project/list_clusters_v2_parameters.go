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

// NewListClustersV2Params creates a new ListClustersV2Params object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListClustersV2Params() *ListClustersV2Params {
	return &ListClustersV2Params{
		timeout: cr.DefaultTimeout,
	}
}

// NewListClustersV2ParamsWithTimeout creates a new ListClustersV2Params object
// with the ability to set a timeout on a request.
func NewListClustersV2ParamsWithTimeout(timeout time.Duration) *ListClustersV2Params {
	return &ListClustersV2Params{
		timeout: timeout,
	}
}

// NewListClustersV2ParamsWithContext creates a new ListClustersV2Params object
// with the ability to set a context for a request.
func NewListClustersV2ParamsWithContext(ctx context.Context) *ListClustersV2Params {
	return &ListClustersV2Params{
		Context: ctx,
	}
}

// NewListClustersV2ParamsWithHTTPClient creates a new ListClustersV2Params object
// with the ability to set a custom HTTPClient for a request.
func NewListClustersV2ParamsWithHTTPClient(client *http.Client) *ListClustersV2Params {
	return &ListClustersV2Params{
		HTTPClient: client,
	}
}

/* ListClustersV2Params contains all the parameters to send to the API endpoint
   for the list clusters v2 operation.

   Typically these are written to a http.Request.
*/
type ListClustersV2Params struct {

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list clusters v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListClustersV2Params) WithDefaults() *ListClustersV2Params {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list clusters v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListClustersV2Params) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list clusters v2 params
func (o *ListClustersV2Params) WithTimeout(timeout time.Duration) *ListClustersV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list clusters v2 params
func (o *ListClustersV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list clusters v2 params
func (o *ListClustersV2Params) WithContext(ctx context.Context) *ListClustersV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list clusters v2 params
func (o *ListClustersV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list clusters v2 params
func (o *ListClustersV2Params) WithHTTPClient(client *http.Client) *ListClustersV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list clusters v2 params
func (o *ListClustersV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithProjectID adds the projectID to the list clusters v2 params
func (o *ListClustersV2Params) WithProjectID(projectID string) *ListClustersV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list clusters v2 params
func (o *ListClustersV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListClustersV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
