// Code generated by go-swagger; DO NOT EDIT.

package allowedregistry

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

// NewCreateAllowedRegistryParams creates a new CreateAllowedRegistryParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewCreateAllowedRegistryParams() *CreateAllowedRegistryParams {
	return &CreateAllowedRegistryParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewCreateAllowedRegistryParamsWithTimeout creates a new CreateAllowedRegistryParams object
// with the ability to set a timeout on a request.
func NewCreateAllowedRegistryParamsWithTimeout(timeout time.Duration) *CreateAllowedRegistryParams {
	return &CreateAllowedRegistryParams{
		timeout: timeout,
	}
}

// NewCreateAllowedRegistryParamsWithContext creates a new CreateAllowedRegistryParams object
// with the ability to set a context for a request.
func NewCreateAllowedRegistryParamsWithContext(ctx context.Context) *CreateAllowedRegistryParams {
	return &CreateAllowedRegistryParams{
		Context: ctx,
	}
}

// NewCreateAllowedRegistryParamsWithHTTPClient creates a new CreateAllowedRegistryParams object
// with the ability to set a custom HTTPClient for a request.
func NewCreateAllowedRegistryParamsWithHTTPClient(client *http.Client) *CreateAllowedRegistryParams {
	return &CreateAllowedRegistryParams{
		HTTPClient: client,
	}
}

/*
CreateAllowedRegistryParams contains all the parameters to send to the API endpoint

	for the create allowed registry operation.

	Typically these are written to a http.Request.
*/
type CreateAllowedRegistryParams struct {

	// Body.
	Body *models.WrBody

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the create allowed registry params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateAllowedRegistryParams) WithDefaults() *CreateAllowedRegistryParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the create allowed registry params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateAllowedRegistryParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the create allowed registry params
func (o *CreateAllowedRegistryParams) WithTimeout(timeout time.Duration) *CreateAllowedRegistryParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the create allowed registry params
func (o *CreateAllowedRegistryParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the create allowed registry params
func (o *CreateAllowedRegistryParams) WithContext(ctx context.Context) *CreateAllowedRegistryParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the create allowed registry params
func (o *CreateAllowedRegistryParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the create allowed registry params
func (o *CreateAllowedRegistryParams) WithHTTPClient(client *http.Client) *CreateAllowedRegistryParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the create allowed registry params
func (o *CreateAllowedRegistryParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the create allowed registry params
func (o *CreateAllowedRegistryParams) WithBody(body *models.WrBody) *CreateAllowedRegistryParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the create allowed registry params
func (o *CreateAllowedRegistryParams) SetBody(body *models.WrBody) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *CreateAllowedRegistryParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
