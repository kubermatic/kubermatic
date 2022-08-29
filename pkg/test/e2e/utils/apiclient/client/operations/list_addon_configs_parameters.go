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
	"github.com/go-openapi/strfmt"
)

// NewListAddonConfigsParams creates a new ListAddonConfigsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListAddonConfigsParams() *ListAddonConfigsParams {
	return &ListAddonConfigsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListAddonConfigsParamsWithTimeout creates a new ListAddonConfigsParams object
// with the ability to set a timeout on a request.
func NewListAddonConfigsParamsWithTimeout(timeout time.Duration) *ListAddonConfigsParams {
	return &ListAddonConfigsParams{
		timeout: timeout,
	}
}

// NewListAddonConfigsParamsWithContext creates a new ListAddonConfigsParams object
// with the ability to set a context for a request.
func NewListAddonConfigsParamsWithContext(ctx context.Context) *ListAddonConfigsParams {
	return &ListAddonConfigsParams{
		Context: ctx,
	}
}

// NewListAddonConfigsParamsWithHTTPClient creates a new ListAddonConfigsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListAddonConfigsParamsWithHTTPClient(client *http.Client) *ListAddonConfigsParams {
	return &ListAddonConfigsParams{
		HTTPClient: client,
	}
}

/*
ListAddonConfigsParams contains all the parameters to send to the API endpoint

	for the list addon configs operation.

	Typically these are written to a http.Request.
*/
type ListAddonConfigsParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list addon configs params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAddonConfigsParams) WithDefaults() *ListAddonConfigsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list addon configs params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAddonConfigsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list addon configs params
func (o *ListAddonConfigsParams) WithTimeout(timeout time.Duration) *ListAddonConfigsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list addon configs params
func (o *ListAddonConfigsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list addon configs params
func (o *ListAddonConfigsParams) WithContext(ctx context.Context) *ListAddonConfigsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list addon configs params
func (o *ListAddonConfigsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list addon configs params
func (o *ListAddonConfigsParams) WithHTTPClient(client *http.Client) *ListAddonConfigsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list addon configs params
func (o *ListAddonConfigsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListAddonConfigsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
