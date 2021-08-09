// Code generated by go-swagger; DO NOT EDIT.

package seed

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

// NewListSeedNamesParams creates a new ListSeedNamesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListSeedNamesParams() *ListSeedNamesParams {
	return &ListSeedNamesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListSeedNamesParamsWithTimeout creates a new ListSeedNamesParams object
// with the ability to set a timeout on a request.
func NewListSeedNamesParamsWithTimeout(timeout time.Duration) *ListSeedNamesParams {
	return &ListSeedNamesParams{
		timeout: timeout,
	}
}

// NewListSeedNamesParamsWithContext creates a new ListSeedNamesParams object
// with the ability to set a context for a request.
func NewListSeedNamesParamsWithContext(ctx context.Context) *ListSeedNamesParams {
	return &ListSeedNamesParams{
		Context: ctx,
	}
}

// NewListSeedNamesParamsWithHTTPClient creates a new ListSeedNamesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListSeedNamesParamsWithHTTPClient(client *http.Client) *ListSeedNamesParams {
	return &ListSeedNamesParams{
		HTTPClient: client,
	}
}

/* ListSeedNamesParams contains all the parameters to send to the API endpoint
   for the list seed names operation.

   Typically these are written to a http.Request.
*/
type ListSeedNamesParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list seed names params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListSeedNamesParams) WithDefaults() *ListSeedNamesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list seed names params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListSeedNamesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list seed names params
func (o *ListSeedNamesParams) WithTimeout(timeout time.Duration) *ListSeedNamesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list seed names params
func (o *ListSeedNamesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list seed names params
func (o *ListSeedNamesParams) WithContext(ctx context.Context) *ListSeedNamesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list seed names params
func (o *ListSeedNamesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list seed names params
func (o *ListSeedNamesParams) WithHTTPClient(client *http.Client) *ListSeedNamesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list seed names params
func (o *ListSeedNamesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListSeedNamesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
