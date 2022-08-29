// Code generated by go-swagger; DO NOT EDIT.

package preset

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
	"github.com/go-openapi/swag"
)

// NewListPresetsParams creates a new ListPresetsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListPresetsParams() *ListPresetsParams {
	return &ListPresetsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListPresetsParamsWithTimeout creates a new ListPresetsParams object
// with the ability to set a timeout on a request.
func NewListPresetsParamsWithTimeout(timeout time.Duration) *ListPresetsParams {
	return &ListPresetsParams{
		timeout: timeout,
	}
}

// NewListPresetsParamsWithContext creates a new ListPresetsParams object
// with the ability to set a context for a request.
func NewListPresetsParamsWithContext(ctx context.Context) *ListPresetsParams {
	return &ListPresetsParams{
		Context: ctx,
	}
}

// NewListPresetsParamsWithHTTPClient creates a new ListPresetsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListPresetsParamsWithHTTPClient(client *http.Client) *ListPresetsParams {
	return &ListPresetsParams{
		HTTPClient: client,
	}
}

/*
ListPresetsParams contains all the parameters to send to the API endpoint

	for the list presets operation.

	Typically these are written to a http.Request.
*/
type ListPresetsParams struct {

	// Disabled.
	Disabled *bool

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list presets params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListPresetsParams) WithDefaults() *ListPresetsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list presets params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListPresetsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list presets params
func (o *ListPresetsParams) WithTimeout(timeout time.Duration) *ListPresetsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list presets params
func (o *ListPresetsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list presets params
func (o *ListPresetsParams) WithContext(ctx context.Context) *ListPresetsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list presets params
func (o *ListPresetsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list presets params
func (o *ListPresetsParams) WithHTTPClient(client *http.Client) *ListPresetsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list presets params
func (o *ListPresetsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithDisabled adds the disabled to the list presets params
func (o *ListPresetsParams) WithDisabled(disabled *bool) *ListPresetsParams {
	o.SetDisabled(disabled)
	return o
}

// SetDisabled adds the disabled to the list presets params
func (o *ListPresetsParams) SetDisabled(disabled *bool) {
	o.Disabled = disabled
}

// WriteToRequest writes these params to a swagger request
func (o *ListPresetsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Disabled != nil {

		// query param disabled
		var qrDisabled bool

		if o.Disabled != nil {
			qrDisabled = *o.Disabled
		}
		qDisabled := swag.FormatBool(qrDisabled)
		if qDisabled != "" {

			if err := r.SetQueryParam("disabled", qDisabled); err != nil {
				return err
			}
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
