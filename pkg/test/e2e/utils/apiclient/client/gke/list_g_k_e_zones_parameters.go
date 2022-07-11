// Code generated by go-swagger; DO NOT EDIT.

package gke

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

// NewListGKEZonesParams creates a new ListGKEZonesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListGKEZonesParams() *ListGKEZonesParams {
	return &ListGKEZonesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListGKEZonesParamsWithTimeout creates a new ListGKEZonesParams object
// with the ability to set a timeout on a request.
func NewListGKEZonesParamsWithTimeout(timeout time.Duration) *ListGKEZonesParams {
	return &ListGKEZonesParams{
		timeout: timeout,
	}
}

// NewListGKEZonesParamsWithContext creates a new ListGKEZonesParams object
// with the ability to set a context for a request.
func NewListGKEZonesParamsWithContext(ctx context.Context) *ListGKEZonesParams {
	return &ListGKEZonesParams{
		Context: ctx,
	}
}

// NewListGKEZonesParamsWithHTTPClient creates a new ListGKEZonesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListGKEZonesParamsWithHTTPClient(client *http.Client) *ListGKEZonesParams {
	return &ListGKEZonesParams{
		HTTPClient: client,
	}
}

/* ListGKEZonesParams contains all the parameters to send to the API endpoint
   for the list g k e zones operation.

   Typically these are written to a http.Request.
*/
type ListGKEZonesParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list g k e zones params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListGKEZonesParams) WithDefaults() *ListGKEZonesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list g k e zones params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListGKEZonesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list g k e zones params
func (o *ListGKEZonesParams) WithTimeout(timeout time.Duration) *ListGKEZonesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list g k e zones params
func (o *ListGKEZonesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list g k e zones params
func (o *ListGKEZonesParams) WithContext(ctx context.Context) *ListGKEZonesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list g k e zones params
func (o *ListGKEZonesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list g k e zones params
func (o *ListGKEZonesParams) WithHTTPClient(client *http.Client) *ListGKEZonesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list g k e zones params
func (o *ListGKEZonesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListGKEZonesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
