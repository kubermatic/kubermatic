// Code generated by go-swagger; DO NOT EDIT.

package resource

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

// NewPatchResourceQuotaParams creates a new PatchResourceQuotaParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewPatchResourceQuotaParams() *PatchResourceQuotaParams {
	return &PatchResourceQuotaParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewPatchResourceQuotaParamsWithTimeout creates a new PatchResourceQuotaParams object
// with the ability to set a timeout on a request.
func NewPatchResourceQuotaParamsWithTimeout(timeout time.Duration) *PatchResourceQuotaParams {
	return &PatchResourceQuotaParams{
		timeout: timeout,
	}
}

// NewPatchResourceQuotaParamsWithContext creates a new PatchResourceQuotaParams object
// with the ability to set a context for a request.
func NewPatchResourceQuotaParamsWithContext(ctx context.Context) *PatchResourceQuotaParams {
	return &PatchResourceQuotaParams{
		Context: ctx,
	}
}

// NewPatchResourceQuotaParamsWithHTTPClient creates a new PatchResourceQuotaParams object
// with the ability to set a custom HTTPClient for a request.
func NewPatchResourceQuotaParamsWithHTTPClient(client *http.Client) *PatchResourceQuotaParams {
	return &PatchResourceQuotaParams{
		HTTPClient: client,
	}
}

/* PatchResourceQuotaParams contains all the parameters to send to the API endpoint
   for the patch resource quota operation.

   Typically these are written to a http.Request.
*/
type PatchResourceQuotaParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the patch resource quota params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *PatchResourceQuotaParams) WithDefaults() *PatchResourceQuotaParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the patch resource quota params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *PatchResourceQuotaParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the patch resource quota params
func (o *PatchResourceQuotaParams) WithTimeout(timeout time.Duration) *PatchResourceQuotaParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the patch resource quota params
func (o *PatchResourceQuotaParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the patch resource quota params
func (o *PatchResourceQuotaParams) WithContext(ctx context.Context) *PatchResourceQuotaParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the patch resource quota params
func (o *PatchResourceQuotaParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the patch resource quota params
func (o *PatchResourceQuotaParams) WithHTTPClient(client *http.Client) *PatchResourceQuotaParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the patch resource quota params
func (o *PatchResourceQuotaParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *PatchResourceQuotaParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
