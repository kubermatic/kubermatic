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

// NewDeleteSeedParams creates a new DeleteSeedParams object
// with the default values initialized.
func NewDeleteSeedParams() *DeleteSeedParams {
	var ()
	return &DeleteSeedParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteSeedParamsWithTimeout creates a new DeleteSeedParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewDeleteSeedParamsWithTimeout(timeout time.Duration) *DeleteSeedParams {
	var ()
	return &DeleteSeedParams{

		timeout: timeout,
	}
}

// NewDeleteSeedParamsWithContext creates a new DeleteSeedParams object
// with the default values initialized, and the ability to set a context for a request
func NewDeleteSeedParamsWithContext(ctx context.Context) *DeleteSeedParams {
	var ()
	return &DeleteSeedParams{

		Context: ctx,
	}
}

// NewDeleteSeedParamsWithHTTPClient creates a new DeleteSeedParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewDeleteSeedParamsWithHTTPClient(client *http.Client) *DeleteSeedParams {
	var ()
	return &DeleteSeedParams{
		HTTPClient: client,
	}
}

/*DeleteSeedParams contains all the parameters to send to the API endpoint
for the delete seed operation typically these are written to a http.Request
*/
type DeleteSeedParams struct {

	/*SeedName*/
	Name string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the delete seed params
func (o *DeleteSeedParams) WithTimeout(timeout time.Duration) *DeleteSeedParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete seed params
func (o *DeleteSeedParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete seed params
func (o *DeleteSeedParams) WithContext(ctx context.Context) *DeleteSeedParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete seed params
func (o *DeleteSeedParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete seed params
func (o *DeleteSeedParams) WithHTTPClient(client *http.Client) *DeleteSeedParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete seed params
func (o *DeleteSeedParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithName adds the seedName to the delete seed params
func (o *DeleteSeedParams) WithName(seedName string) *DeleteSeedParams {
	o.SetName(seedName)
	return o
}

// SetName adds the seedName to the delete seed params
func (o *DeleteSeedParams) SetName(seedName string) {
	o.Name = seedName
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteSeedParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param seed_name
	if err := r.SetPathParam("seed_name", o.Name); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
