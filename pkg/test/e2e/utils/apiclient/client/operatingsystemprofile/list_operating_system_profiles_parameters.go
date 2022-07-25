// Code generated by go-swagger; DO NOT EDIT.

package operatingsystemprofile

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

// NewListOperatingSystemProfilesParams creates a new ListOperatingSystemProfilesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListOperatingSystemProfilesParams() *ListOperatingSystemProfilesParams {
	return &ListOperatingSystemProfilesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListOperatingSystemProfilesParamsWithTimeout creates a new ListOperatingSystemProfilesParams object
// with the ability to set a timeout on a request.
func NewListOperatingSystemProfilesParamsWithTimeout(timeout time.Duration) *ListOperatingSystemProfilesParams {
	return &ListOperatingSystemProfilesParams{
		timeout: timeout,
	}
}

// NewListOperatingSystemProfilesParamsWithContext creates a new ListOperatingSystemProfilesParams object
// with the ability to set a context for a request.
func NewListOperatingSystemProfilesParamsWithContext(ctx context.Context) *ListOperatingSystemProfilesParams {
	return &ListOperatingSystemProfilesParams{
		Context: ctx,
	}
}

// NewListOperatingSystemProfilesParamsWithHTTPClient creates a new ListOperatingSystemProfilesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListOperatingSystemProfilesParamsWithHTTPClient(client *http.Client) *ListOperatingSystemProfilesParams {
	return &ListOperatingSystemProfilesParams{
		HTTPClient: client,
	}
}

/* ListOperatingSystemProfilesParams contains all the parameters to send to the API endpoint
   for the list operating system profiles operation.

   Typically these are written to a http.Request.
*/
type ListOperatingSystemProfilesParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list operating system profiles params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListOperatingSystemProfilesParams) WithDefaults() *ListOperatingSystemProfilesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list operating system profiles params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListOperatingSystemProfilesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list operating system profiles params
func (o *ListOperatingSystemProfilesParams) WithTimeout(timeout time.Duration) *ListOperatingSystemProfilesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list operating system profiles params
func (o *ListOperatingSystemProfilesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list operating system profiles params
func (o *ListOperatingSystemProfilesParams) WithContext(ctx context.Context) *ListOperatingSystemProfilesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list operating system profiles params
func (o *ListOperatingSystemProfilesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list operating system profiles params
func (o *ListOperatingSystemProfilesParams) WithHTTPClient(client *http.Client) *ListOperatingSystemProfilesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list operating system profiles params
func (o *ListOperatingSystemProfilesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListOperatingSystemProfilesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
