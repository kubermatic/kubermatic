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

// NewDeleteProjectParams creates a new DeleteProjectParams object
// with the default values initialized.
func NewDeleteProjectParams() *DeleteProjectParams {
	var ()
	return &DeleteProjectParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteProjectParamsWithTimeout creates a new DeleteProjectParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewDeleteProjectParamsWithTimeout(timeout time.Duration) *DeleteProjectParams {
	var ()
	return &DeleteProjectParams{

		timeout: timeout,
	}
}

// NewDeleteProjectParamsWithContext creates a new DeleteProjectParams object
// with the default values initialized, and the ability to set a context for a request
func NewDeleteProjectParamsWithContext(ctx context.Context) *DeleteProjectParams {
	var ()
	return &DeleteProjectParams{

		Context: ctx,
	}
}

// NewDeleteProjectParamsWithHTTPClient creates a new DeleteProjectParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewDeleteProjectParamsWithHTTPClient(client *http.Client) *DeleteProjectParams {
	var ()
	return &DeleteProjectParams{
		HTTPClient: client,
	}
}

/*DeleteProjectParams contains all the parameters to send to the API endpoint
for the delete project operation typically these are written to a http.Request
*/
type DeleteProjectParams struct {

	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the delete project params
func (o *DeleteProjectParams) WithTimeout(timeout time.Duration) *DeleteProjectParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete project params
func (o *DeleteProjectParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete project params
func (o *DeleteProjectParams) WithContext(ctx context.Context) *DeleteProjectParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete project params
func (o *DeleteProjectParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete project params
func (o *DeleteProjectParams) WithHTTPClient(client *http.Client) *DeleteProjectParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete project params
func (o *DeleteProjectParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithProjectID adds the projectID to the delete project params
func (o *DeleteProjectParams) WithProjectID(projectID string) *DeleteProjectParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the delete project params
func (o *DeleteProjectParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteProjectParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
