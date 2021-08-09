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

// NewAssignSSHKeyToClusterV2Params creates a new AssignSSHKeyToClusterV2Params object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewAssignSSHKeyToClusterV2Params() *AssignSSHKeyToClusterV2Params {
	return &AssignSSHKeyToClusterV2Params{
		timeout: cr.DefaultTimeout,
	}
}

// NewAssignSSHKeyToClusterV2ParamsWithTimeout creates a new AssignSSHKeyToClusterV2Params object
// with the ability to set a timeout on a request.
func NewAssignSSHKeyToClusterV2ParamsWithTimeout(timeout time.Duration) *AssignSSHKeyToClusterV2Params {
	return &AssignSSHKeyToClusterV2Params{
		timeout: timeout,
	}
}

// NewAssignSSHKeyToClusterV2ParamsWithContext creates a new AssignSSHKeyToClusterV2Params object
// with the ability to set a context for a request.
func NewAssignSSHKeyToClusterV2ParamsWithContext(ctx context.Context) *AssignSSHKeyToClusterV2Params {
	return &AssignSSHKeyToClusterV2Params{
		Context: ctx,
	}
}

// NewAssignSSHKeyToClusterV2ParamsWithHTTPClient creates a new AssignSSHKeyToClusterV2Params object
// with the ability to set a custom HTTPClient for a request.
func NewAssignSSHKeyToClusterV2ParamsWithHTTPClient(client *http.Client) *AssignSSHKeyToClusterV2Params {
	return &AssignSSHKeyToClusterV2Params{
		HTTPClient: client,
	}
}

/* AssignSSHKeyToClusterV2Params contains all the parameters to send to the API endpoint
   for the assign SSH key to cluster v2 operation.

   Typically these are written to a http.Request.
*/
type AssignSSHKeyToClusterV2Params struct {

	// ClusterID.
	ClusterID string

	// KeyID.
	KeyID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the assign SSH key to cluster v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AssignSSHKeyToClusterV2Params) WithDefaults() *AssignSSHKeyToClusterV2Params {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the assign SSH key to cluster v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *AssignSSHKeyToClusterV2Params) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) WithTimeout(timeout time.Duration) *AssignSSHKeyToClusterV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) WithContext(ctx context.Context) *AssignSSHKeyToClusterV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) WithHTTPClient(client *http.Client) *AssignSSHKeyToClusterV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) WithClusterID(clusterID string) *AssignSSHKeyToClusterV2Params {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithKeyID adds the keyID to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) WithKeyID(keyID string) *AssignSSHKeyToClusterV2Params {
	o.SetKeyID(keyID)
	return o
}

// SetKeyID adds the keyId to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) SetKeyID(keyID string) {
	o.KeyID = keyID
}

// WithProjectID adds the projectID to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) WithProjectID(projectID string) *AssignSSHKeyToClusterV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the assign SSH key to cluster v2 params
func (o *AssignSSHKeyToClusterV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *AssignSSHKeyToClusterV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
		return err
	}

	// path param key_id
	if err := r.SetPathParam("key_id", o.KeyID); err != nil {
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
