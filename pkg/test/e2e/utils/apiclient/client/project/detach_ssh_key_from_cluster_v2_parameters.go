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

// NewDetachSSHKeyFromClusterV2Params creates a new DetachSSHKeyFromClusterV2Params object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDetachSSHKeyFromClusterV2Params() *DetachSSHKeyFromClusterV2Params {
	return &DetachSSHKeyFromClusterV2Params{
		timeout: cr.DefaultTimeout,
	}
}

// NewDetachSSHKeyFromClusterV2ParamsWithTimeout creates a new DetachSSHKeyFromClusterV2Params object
// with the ability to set a timeout on a request.
func NewDetachSSHKeyFromClusterV2ParamsWithTimeout(timeout time.Duration) *DetachSSHKeyFromClusterV2Params {
	return &DetachSSHKeyFromClusterV2Params{
		timeout: timeout,
	}
}

// NewDetachSSHKeyFromClusterV2ParamsWithContext creates a new DetachSSHKeyFromClusterV2Params object
// with the ability to set a context for a request.
func NewDetachSSHKeyFromClusterV2ParamsWithContext(ctx context.Context) *DetachSSHKeyFromClusterV2Params {
	return &DetachSSHKeyFromClusterV2Params{
		Context: ctx,
	}
}

// NewDetachSSHKeyFromClusterV2ParamsWithHTTPClient creates a new DetachSSHKeyFromClusterV2Params object
// with the ability to set a custom HTTPClient for a request.
func NewDetachSSHKeyFromClusterV2ParamsWithHTTPClient(client *http.Client) *DetachSSHKeyFromClusterV2Params {
	return &DetachSSHKeyFromClusterV2Params{
		HTTPClient: client,
	}
}

/* DetachSSHKeyFromClusterV2Params contains all the parameters to send to the API endpoint
   for the detach SSH key from cluster v2 operation.

   Typically these are written to a http.Request.
*/
type DetachSSHKeyFromClusterV2Params struct {

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

// WithDefaults hydrates default values in the detach SSH key from cluster v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DetachSSHKeyFromClusterV2Params) WithDefaults() *DetachSSHKeyFromClusterV2Params {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the detach SSH key from cluster v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DetachSSHKeyFromClusterV2Params) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) WithTimeout(timeout time.Duration) *DetachSSHKeyFromClusterV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) WithContext(ctx context.Context) *DetachSSHKeyFromClusterV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) WithHTTPClient(client *http.Client) *DetachSSHKeyFromClusterV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) WithClusterID(clusterID string) *DetachSSHKeyFromClusterV2Params {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithKeyID adds the keyID to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) WithKeyID(keyID string) *DetachSSHKeyFromClusterV2Params {
	o.SetKeyID(keyID)
	return o
}

// SetKeyID adds the keyId to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) SetKeyID(keyID string) {
	o.KeyID = keyID
}

// WithProjectID adds the projectID to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) WithProjectID(projectID string) *DetachSSHKeyFromClusterV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the detach SSH key from cluster v2 params
func (o *DetachSSHKeyFromClusterV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *DetachSSHKeyFromClusterV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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