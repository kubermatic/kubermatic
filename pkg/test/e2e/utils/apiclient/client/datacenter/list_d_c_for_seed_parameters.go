// Code generated by go-swagger; DO NOT EDIT.

package datacenter

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

// NewListDCForSeedParams creates a new ListDCForSeedParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListDCForSeedParams() *ListDCForSeedParams {
	return &ListDCForSeedParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListDCForSeedParamsWithTimeout creates a new ListDCForSeedParams object
// with the ability to set a timeout on a request.
func NewListDCForSeedParamsWithTimeout(timeout time.Duration) *ListDCForSeedParams {
	return &ListDCForSeedParams{
		timeout: timeout,
	}
}

// NewListDCForSeedParamsWithContext creates a new ListDCForSeedParams object
// with the ability to set a context for a request.
func NewListDCForSeedParamsWithContext(ctx context.Context) *ListDCForSeedParams {
	return &ListDCForSeedParams{
		Context: ctx,
	}
}

// NewListDCForSeedParamsWithHTTPClient creates a new ListDCForSeedParams object
// with the ability to set a custom HTTPClient for a request.
func NewListDCForSeedParamsWithHTTPClient(client *http.Client) *ListDCForSeedParams {
	return &ListDCForSeedParams{
		HTTPClient: client,
	}
}

/* ListDCForSeedParams contains all the parameters to send to the API endpoint
   for the list d c for seed operation.

   Typically these are written to a http.Request.
*/
type ListDCForSeedParams struct {

	// SeedName.
	Seed string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list d c for seed params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListDCForSeedParams) WithDefaults() *ListDCForSeedParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list d c for seed params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListDCForSeedParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list d c for seed params
func (o *ListDCForSeedParams) WithTimeout(timeout time.Duration) *ListDCForSeedParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list d c for seed params
func (o *ListDCForSeedParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list d c for seed params
func (o *ListDCForSeedParams) WithContext(ctx context.Context) *ListDCForSeedParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list d c for seed params
func (o *ListDCForSeedParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list d c for seed params
func (o *ListDCForSeedParams) WithHTTPClient(client *http.Client) *ListDCForSeedParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list d c for seed params
func (o *ListDCForSeedParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithSeed adds the seedName to the list d c for seed params
func (o *ListDCForSeedParams) WithSeed(seedName string) *ListDCForSeedParams {
	o.SetSeed(seedName)
	return o
}

// SetSeed adds the seedName to the list d c for seed params
func (o *ListDCForSeedParams) SetSeed(seedName string) {
	o.Seed = seedName
}

// WriteToRequest writes these params to a swagger request
func (o *ListDCForSeedParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param seed_name
	if err := r.SetPathParam("seed_name", o.Seed); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
