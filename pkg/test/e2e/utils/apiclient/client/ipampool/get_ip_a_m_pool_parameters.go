// Code generated by go-swagger; DO NOT EDIT.

package ipampool

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

// NewGetIPAMPoolParams creates a new GetIPAMPoolParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetIPAMPoolParams() *GetIPAMPoolParams {
	return &GetIPAMPoolParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetIPAMPoolParamsWithTimeout creates a new GetIPAMPoolParams object
// with the ability to set a timeout on a request.
func NewGetIPAMPoolParamsWithTimeout(timeout time.Duration) *GetIPAMPoolParams {
	return &GetIPAMPoolParams{
		timeout: timeout,
	}
}

// NewGetIPAMPoolParamsWithContext creates a new GetIPAMPoolParams object
// with the ability to set a context for a request.
func NewGetIPAMPoolParamsWithContext(ctx context.Context) *GetIPAMPoolParams {
	return &GetIPAMPoolParams{
		Context: ctx,
	}
}

// NewGetIPAMPoolParamsWithHTTPClient creates a new GetIPAMPoolParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetIPAMPoolParamsWithHTTPClient(client *http.Client) *GetIPAMPoolParams {
	return &GetIPAMPoolParams{
		HTTPClient: client,
	}
}

/* GetIPAMPoolParams contains all the parameters to send to the API endpoint
   for the get IP a m pool operation.

   Typically these are written to a http.Request.
*/
type GetIPAMPoolParams struct {

	// IpampoolName.
	IPAMPoolName string

	// SeedName.
	SeedName string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get IP a m pool params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetIPAMPoolParams) WithDefaults() *GetIPAMPoolParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get IP a m pool params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetIPAMPoolParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get IP a m pool params
func (o *GetIPAMPoolParams) WithTimeout(timeout time.Duration) *GetIPAMPoolParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get IP a m pool params
func (o *GetIPAMPoolParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get IP a m pool params
func (o *GetIPAMPoolParams) WithContext(ctx context.Context) *GetIPAMPoolParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get IP a m pool params
func (o *GetIPAMPoolParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get IP a m pool params
func (o *GetIPAMPoolParams) WithHTTPClient(client *http.Client) *GetIPAMPoolParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get IP a m pool params
func (o *GetIPAMPoolParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithIPAMPoolName adds the ipampoolName to the get IP a m pool params
func (o *GetIPAMPoolParams) WithIPAMPoolName(ipampoolName string) *GetIPAMPoolParams {
	o.SetIPAMPoolName(ipampoolName)
	return o
}

// SetIPAMPoolName adds the ipampoolName to the get IP a m pool params
func (o *GetIPAMPoolParams) SetIPAMPoolName(ipampoolName string) {
	o.IPAMPoolName = ipampoolName
}

// WithSeedName adds the seedName to the get IP a m pool params
func (o *GetIPAMPoolParams) WithSeedName(seedName string) *GetIPAMPoolParams {
	o.SetSeedName(seedName)
	return o
}

// SetSeedName adds the seedName to the get IP a m pool params
func (o *GetIPAMPoolParams) SetSeedName(seedName string) {
	o.SeedName = seedName
}

// WriteToRequest writes these params to a swagger request
func (o *GetIPAMPoolParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param ipampool_name
	if err := r.SetPathParam("ipampool_name", o.IPAMPoolName); err != nil {
		return err
	}

	// path param seed_name
	if err := r.SetPathParam("seed_name", o.SeedName); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
