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

// NewGetProjectParams creates a new GetProjectParams object
// with the default values initialized.
func NewGetProjectParams() *GetProjectParams {
	var ()
	return &GetProjectParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetProjectParamsWithTimeout creates a new GetProjectParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetProjectParamsWithTimeout(timeout time.Duration) *GetProjectParams {
	var ()
	return &GetProjectParams{

		timeout: timeout,
	}
}

// NewGetProjectParamsWithContext creates a new GetProjectParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetProjectParamsWithContext(ctx context.Context) *GetProjectParams {
	var ()
	return &GetProjectParams{

		Context: ctx,
	}
}

// NewGetProjectParamsWithHTTPClient creates a new GetProjectParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetProjectParamsWithHTTPClient(client *http.Client) *GetProjectParams {
	var ()
	return &GetProjectParams{
		HTTPClient: client,
	}
}

/*GetProjectParams contains all the parameters to send to the API endpoint
for the get project operation typically these are written to a http.Request
*/
type GetProjectParams struct {

	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get project params
func (o *GetProjectParams) WithTimeout(timeout time.Duration) *GetProjectParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get project params
func (o *GetProjectParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get project params
func (o *GetProjectParams) WithContext(ctx context.Context) *GetProjectParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get project params
func (o *GetProjectParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get project params
func (o *GetProjectParams) WithHTTPClient(client *http.Client) *GetProjectParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get project params
func (o *GetProjectParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithProjectID adds the projectID to the get project params
func (o *GetProjectParams) WithProjectID(projectID string) *GetProjectParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get project params
func (o *GetProjectParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *GetProjectParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
