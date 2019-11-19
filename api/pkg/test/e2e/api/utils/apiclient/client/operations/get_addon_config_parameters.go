// Code generated by go-swagger; DO NOT EDIT.

package operations

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

// NewGetAddonConfigParams creates a new GetAddonConfigParams object
// with the default values initialized.
func NewGetAddonConfigParams() *GetAddonConfigParams {
	var ()
	return &GetAddonConfigParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetAddonConfigParamsWithTimeout creates a new GetAddonConfigParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetAddonConfigParamsWithTimeout(timeout time.Duration) *GetAddonConfigParams {
	var ()
	return &GetAddonConfigParams{

		timeout: timeout,
	}
}

// NewGetAddonConfigParamsWithContext creates a new GetAddonConfigParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetAddonConfigParamsWithContext(ctx context.Context) *GetAddonConfigParams {
	var ()
	return &GetAddonConfigParams{

		Context: ctx,
	}
}

// NewGetAddonConfigParamsWithHTTPClient creates a new GetAddonConfigParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetAddonConfigParamsWithHTTPClient(client *http.Client) *GetAddonConfigParams {
	var ()
	return &GetAddonConfigParams{
		HTTPClient: client,
	}
}

/*GetAddonConfigParams contains all the parameters to send to the API endpoint
for the get addon config operation typically these are written to a http.Request
*/
type GetAddonConfigParams struct {

	/*AddonID*/
	AddonID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get addon config params
func (o *GetAddonConfigParams) WithTimeout(timeout time.Duration) *GetAddonConfigParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get addon config params
func (o *GetAddonConfigParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get addon config params
func (o *GetAddonConfigParams) WithContext(ctx context.Context) *GetAddonConfigParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get addon config params
func (o *GetAddonConfigParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get addon config params
func (o *GetAddonConfigParams) WithHTTPClient(client *http.Client) *GetAddonConfigParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get addon config params
func (o *GetAddonConfigParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAddonID adds the addonID to the get addon config params
func (o *GetAddonConfigParams) WithAddonID(addonID string) *GetAddonConfigParams {
	o.SetAddonID(addonID)
	return o
}

// SetAddonID adds the addonId to the get addon config params
func (o *GetAddonConfigParams) SetAddonID(addonID string) {
	o.AddonID = addonID
}

// WriteToRequest writes these params to a swagger request
func (o *GetAddonConfigParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param addon_id
	if err := r.SetPathParam("addon_id", o.AddonID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
