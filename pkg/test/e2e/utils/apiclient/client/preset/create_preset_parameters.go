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

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// NewCreatePresetParams creates a new CreatePresetParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewCreatePresetParams() *CreatePresetParams {
	return &CreatePresetParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewCreatePresetParamsWithTimeout creates a new CreatePresetParams object
// with the ability to set a timeout on a request.
func NewCreatePresetParamsWithTimeout(timeout time.Duration) *CreatePresetParams {
	return &CreatePresetParams{
		timeout: timeout,
	}
}

// NewCreatePresetParamsWithContext creates a new CreatePresetParams object
// with the ability to set a context for a request.
func NewCreatePresetParamsWithContext(ctx context.Context) *CreatePresetParams {
	return &CreatePresetParams{
		Context: ctx,
	}
}

// NewCreatePresetParamsWithHTTPClient creates a new CreatePresetParams object
// with the ability to set a custom HTTPClient for a request.
func NewCreatePresetParamsWithHTTPClient(client *http.Client) *CreatePresetParams {
	return &CreatePresetParams{
		HTTPClient: client,
	}
}

/* CreatePresetParams contains all the parameters to send to the API endpoint
   for the create preset operation.

   Typically these are written to a http.Request.
*/
type CreatePresetParams struct {

	// Body.
	Body *models.Preset

	// ProviderName.
	ProviderName string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the create preset params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreatePresetParams) WithDefaults() *CreatePresetParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the create preset params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreatePresetParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the create preset params
func (o *CreatePresetParams) WithTimeout(timeout time.Duration) *CreatePresetParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the create preset params
func (o *CreatePresetParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the create preset params
func (o *CreatePresetParams) WithContext(ctx context.Context) *CreatePresetParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the create preset params
func (o *CreatePresetParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the create preset params
func (o *CreatePresetParams) WithHTTPClient(client *http.Client) *CreatePresetParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the create preset params
func (o *CreatePresetParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the create preset params
func (o *CreatePresetParams) WithBody(body *models.Preset) *CreatePresetParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the create preset params
func (o *CreatePresetParams) SetBody(body *models.Preset) {
	o.Body = body
}

// WithProviderName adds the providerName to the create preset params
func (o *CreatePresetParams) WithProviderName(providerName string) *CreatePresetParams {
	o.SetProviderName(providerName)
	return o
}

// SetProviderName adds the providerName to the create preset params
func (o *CreatePresetParams) SetProviderName(providerName string) {
	o.ProviderName = providerName
}

// WriteToRequest writes these params to a swagger request
func (o *CreatePresetParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	// path param provider_name
	if err := r.SetPathParam("provider_name", o.ProviderName); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
