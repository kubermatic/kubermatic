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

// NewDeleteIPAMPoolParams creates a new DeleteIPAMPoolParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteIPAMPoolParams() *DeleteIPAMPoolParams {
	return &DeleteIPAMPoolParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteIPAMPoolParamsWithTimeout creates a new DeleteIPAMPoolParams object
// with the ability to set a timeout on a request.
func NewDeleteIPAMPoolParamsWithTimeout(timeout time.Duration) *DeleteIPAMPoolParams {
	return &DeleteIPAMPoolParams{
		timeout: timeout,
	}
}

// NewDeleteIPAMPoolParamsWithContext creates a new DeleteIPAMPoolParams object
// with the ability to set a context for a request.
func NewDeleteIPAMPoolParamsWithContext(ctx context.Context) *DeleteIPAMPoolParams {
	return &DeleteIPAMPoolParams{
		Context: ctx,
	}
}

// NewDeleteIPAMPoolParamsWithHTTPClient creates a new DeleteIPAMPoolParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteIPAMPoolParamsWithHTTPClient(client *http.Client) *DeleteIPAMPoolParams {
	return &DeleteIPAMPoolParams{
		HTTPClient: client,
	}
}

/* DeleteIPAMPoolParams contains all the parameters to send to the API endpoint
   for the delete IP a m pool operation.

   Typically these are written to a http.Request.
*/
type DeleteIPAMPoolParams struct {

	// IpampoolName.
	IPAMPoolName string

	// SeedName.
	SeedName string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete IP a m pool params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteIPAMPoolParams) WithDefaults() *DeleteIPAMPoolParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete IP a m pool params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteIPAMPoolParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete IP a m pool params
func (o *DeleteIPAMPoolParams) WithTimeout(timeout time.Duration) *DeleteIPAMPoolParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete IP a m pool params
func (o *DeleteIPAMPoolParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete IP a m pool params
func (o *DeleteIPAMPoolParams) WithContext(ctx context.Context) *DeleteIPAMPoolParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete IP a m pool params
func (o *DeleteIPAMPoolParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete IP a m pool params
func (o *DeleteIPAMPoolParams) WithHTTPClient(client *http.Client) *DeleteIPAMPoolParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete IP a m pool params
func (o *DeleteIPAMPoolParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithIPAMPoolName adds the ipampoolName to the delete IP a m pool params
func (o *DeleteIPAMPoolParams) WithIPAMPoolName(ipampoolName string) *DeleteIPAMPoolParams {
	o.SetIPAMPoolName(ipampoolName)
	return o
}

// SetIPAMPoolName adds the ipampoolName to the delete IP a m pool params
func (o *DeleteIPAMPoolParams) SetIPAMPoolName(ipampoolName string) {
	o.IPAMPoolName = ipampoolName
}

// WithSeedName adds the seedName to the delete IP a m pool params
func (o *DeleteIPAMPoolParams) WithSeedName(seedName string) *DeleteIPAMPoolParams {
	o.SetSeedName(seedName)
	return o
}

// SetSeedName adds the seedName to the delete IP a m pool params
func (o *DeleteIPAMPoolParams) SetSeedName(seedName string) {
	o.SeedName = seedName
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteIPAMPoolParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
