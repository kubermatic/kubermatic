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
	"github.com/go-openapi/strfmt"
)

// NewUpdateSeedParams creates a new UpdateSeedParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewUpdateSeedParams() *UpdateSeedParams {
	return &UpdateSeedParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewUpdateSeedParamsWithTimeout creates a new UpdateSeedParams object
// with the ability to set a timeout on a request.
func NewUpdateSeedParamsWithTimeout(timeout time.Duration) *UpdateSeedParams {
	return &UpdateSeedParams{
		timeout: timeout,
	}
}

// NewUpdateSeedParamsWithContext creates a new UpdateSeedParams object
// with the ability to set a context for a request.
func NewUpdateSeedParamsWithContext(ctx context.Context) *UpdateSeedParams {
	return &UpdateSeedParams{
		Context: ctx,
	}
}

// NewUpdateSeedParamsWithHTTPClient creates a new UpdateSeedParams object
// with the ability to set a custom HTTPClient for a request.
func NewUpdateSeedParamsWithHTTPClient(client *http.Client) *UpdateSeedParams {
	return &UpdateSeedParams{
		HTTPClient: client,
	}
}

/*
UpdateSeedParams contains all the parameters to send to the API endpoint

	for the update seed operation.

	Typically these are written to a http.Request.
*/
type UpdateSeedParams struct {

	// Body.
	Body UpdateSeedBody

	// SeedName.
	Name string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the update seed params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpdateSeedParams) WithDefaults() *UpdateSeedParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the update seed params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpdateSeedParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the update seed params
func (o *UpdateSeedParams) WithTimeout(timeout time.Duration) *UpdateSeedParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the update seed params
func (o *UpdateSeedParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the update seed params
func (o *UpdateSeedParams) WithContext(ctx context.Context) *UpdateSeedParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the update seed params
func (o *UpdateSeedParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the update seed params
func (o *UpdateSeedParams) WithHTTPClient(client *http.Client) *UpdateSeedParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the update seed params
func (o *UpdateSeedParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the update seed params
func (o *UpdateSeedParams) WithBody(body UpdateSeedBody) *UpdateSeedParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the update seed params
func (o *UpdateSeedParams) SetBody(body UpdateSeedBody) {
	o.Body = body
}

// WithName adds the seedName to the update seed params
func (o *UpdateSeedParams) WithName(seedName string) *UpdateSeedParams {
	o.SetName(seedName)
	return o
}

// SetName adds the seedName to the update seed params
func (o *UpdateSeedParams) SetName(seedName string) {
	o.Name = seedName
}

// WriteToRequest writes these params to a swagger request
func (o *UpdateSeedParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if err := r.SetBodyParam(o.Body); err != nil {
		return err
	}

	// path param seed_name
	if err := r.SetPathParam("seed_name", o.Name); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
