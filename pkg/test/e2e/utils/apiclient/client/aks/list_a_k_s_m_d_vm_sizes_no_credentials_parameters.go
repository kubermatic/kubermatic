// Code generated by go-swagger; DO NOT EDIT.

package aks

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

// NewListAKSMDVMSizesNoCredentialsParams creates a new ListAKSMDVMSizesNoCredentialsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListAKSMDVMSizesNoCredentialsParams() *ListAKSMDVMSizesNoCredentialsParams {
	return &ListAKSMDVMSizesNoCredentialsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListAKSMDVMSizesNoCredentialsParamsWithTimeout creates a new ListAKSMDVMSizesNoCredentialsParams object
// with the ability to set a timeout on a request.
func NewListAKSMDVMSizesNoCredentialsParamsWithTimeout(timeout time.Duration) *ListAKSMDVMSizesNoCredentialsParams {
	return &ListAKSMDVMSizesNoCredentialsParams{
		timeout: timeout,
	}
}

// NewListAKSMDVMSizesNoCredentialsParamsWithContext creates a new ListAKSMDVMSizesNoCredentialsParams object
// with the ability to set a context for a request.
func NewListAKSMDVMSizesNoCredentialsParamsWithContext(ctx context.Context) *ListAKSMDVMSizesNoCredentialsParams {
	return &ListAKSMDVMSizesNoCredentialsParams{
		Context: ctx,
	}
}

// NewListAKSMDVMSizesNoCredentialsParamsWithHTTPClient creates a new ListAKSMDVMSizesNoCredentialsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListAKSMDVMSizesNoCredentialsParamsWithHTTPClient(client *http.Client) *ListAKSMDVMSizesNoCredentialsParams {
	return &ListAKSMDVMSizesNoCredentialsParams{
		HTTPClient: client,
	}
}

/* ListAKSMDVMSizesNoCredentialsParams contains all the parameters to send to the API endpoint
   for the list a k s m d VM sizes no credentials operation.

   Typically these are written to a http.Request.
*/
type ListAKSMDVMSizesNoCredentialsParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list a k s m d VM sizes no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAKSMDVMSizesNoCredentialsParams) WithDefaults() *ListAKSMDVMSizesNoCredentialsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list a k s m d VM sizes no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAKSMDVMSizesNoCredentialsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list a k s m d VM sizes no credentials params
func (o *ListAKSMDVMSizesNoCredentialsParams) WithTimeout(timeout time.Duration) *ListAKSMDVMSizesNoCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list a k s m d VM sizes no credentials params
func (o *ListAKSMDVMSizesNoCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list a k s m d VM sizes no credentials params
func (o *ListAKSMDVMSizesNoCredentialsParams) WithContext(ctx context.Context) *ListAKSMDVMSizesNoCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list a k s m d VM sizes no credentials params
func (o *ListAKSMDVMSizesNoCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list a k s m d VM sizes no credentials params
func (o *ListAKSMDVMSizesNoCredentialsParams) WithHTTPClient(client *http.Client) *ListAKSMDVMSizesNoCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list a k s m d VM sizes no credentials params
func (o *ListAKSMDVMSizesNoCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListAKSMDVMSizesNoCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
