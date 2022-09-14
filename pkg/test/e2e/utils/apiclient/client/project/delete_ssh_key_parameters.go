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

// NewDeleteSSHKeyParams creates a new DeleteSSHKeyParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteSSHKeyParams() *DeleteSSHKeyParams {
	return &DeleteSSHKeyParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteSSHKeyParamsWithTimeout creates a new DeleteSSHKeyParams object
// with the ability to set a timeout on a request.
func NewDeleteSSHKeyParamsWithTimeout(timeout time.Duration) *DeleteSSHKeyParams {
	return &DeleteSSHKeyParams{
		timeout: timeout,
	}
}

// NewDeleteSSHKeyParamsWithContext creates a new DeleteSSHKeyParams object
// with the ability to set a context for a request.
func NewDeleteSSHKeyParamsWithContext(ctx context.Context) *DeleteSSHKeyParams {
	return &DeleteSSHKeyParams{
		Context: ctx,
	}
}

// NewDeleteSSHKeyParamsWithHTTPClient creates a new DeleteSSHKeyParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteSSHKeyParamsWithHTTPClient(client *http.Client) *DeleteSSHKeyParams {
	return &DeleteSSHKeyParams{
		HTTPClient: client,
	}
}

/*
DeleteSSHKeyParams contains all the parameters to send to the API endpoint

	for the delete SSH key operation.

	Typically these are written to a http.Request.
*/
type DeleteSSHKeyParams struct {

	// KeyID.
	SSHKeyID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete SSH key params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteSSHKeyParams) WithDefaults() *DeleteSSHKeyParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete SSH key params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteSSHKeyParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete SSH key params
func (o *DeleteSSHKeyParams) WithTimeout(timeout time.Duration) *DeleteSSHKeyParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete SSH key params
func (o *DeleteSSHKeyParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete SSH key params
func (o *DeleteSSHKeyParams) WithContext(ctx context.Context) *DeleteSSHKeyParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete SSH key params
func (o *DeleteSSHKeyParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete SSH key params
func (o *DeleteSSHKeyParams) WithHTTPClient(client *http.Client) *DeleteSSHKeyParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete SSH key params
func (o *DeleteSSHKeyParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithSSHKeyID adds the keyID to the delete SSH key params
func (o *DeleteSSHKeyParams) WithSSHKeyID(keyID string) *DeleteSSHKeyParams {
	o.SetSSHKeyID(keyID)
	return o
}

// SetSSHKeyID adds the keyId to the delete SSH key params
func (o *DeleteSSHKeyParams) SetSSHKeyID(keyID string) {
	o.SSHKeyID = keyID
}

// WithProjectID adds the projectID to the delete SSH key params
func (o *DeleteSSHKeyParams) WithProjectID(projectID string) *DeleteSSHKeyParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the delete SSH key params
func (o *DeleteSSHKeyParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteSSHKeyParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param key_id
	if err := r.SetPathParam("key_id", o.SSHKeyID); err != nil {
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
