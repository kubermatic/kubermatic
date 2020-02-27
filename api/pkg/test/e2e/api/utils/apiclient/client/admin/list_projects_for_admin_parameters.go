// Code generated by go-swagger; DO NOT EDIT.

package admin

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"

	strfmt "github.com/go-openapi/strfmt"
)

// NewListProjectsForAdminParams creates a new ListProjectsForAdminParams object
// with the default values initialized.
func NewListProjectsForAdminParams() *ListProjectsForAdminParams {

	return &ListProjectsForAdminParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewListProjectsForAdminParamsWithTimeout creates a new ListProjectsForAdminParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListProjectsForAdminParamsWithTimeout(timeout time.Duration) *ListProjectsForAdminParams {

	return &ListProjectsForAdminParams{

		timeout: timeout,
	}
}

// NewListProjectsForAdminParamsWithContext creates a new ListProjectsForAdminParams object
// with the default values initialized, and the ability to set a context for a request
func NewListProjectsForAdminParamsWithContext(ctx context.Context) *ListProjectsForAdminParams {

	return &ListProjectsForAdminParams{

		Context: ctx,
	}
}

// NewListProjectsForAdminParamsWithHTTPClient creates a new ListProjectsForAdminParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListProjectsForAdminParamsWithHTTPClient(client *http.Client) *ListProjectsForAdminParams {

	return &ListProjectsForAdminParams{
		HTTPClient: client,
	}
}

/*ListProjectsForAdminParams contains all the parameters to send to the API endpoint
for the list projects for admin operation typically these are written to a http.Request
*/
type ListProjectsForAdminParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list projects for admin params
func (o *ListProjectsForAdminParams) WithTimeout(timeout time.Duration) *ListProjectsForAdminParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list projects for admin params
func (o *ListProjectsForAdminParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list projects for admin params
func (o *ListProjectsForAdminParams) WithContext(ctx context.Context) *ListProjectsForAdminParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list projects for admin params
func (o *ListProjectsForAdminParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list projects for admin params
func (o *ListProjectsForAdminParams) WithHTTPClient(client *http.Client) *ListProjectsForAdminParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list projects for admin params
func (o *ListProjectsForAdminParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListProjectsForAdminParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
