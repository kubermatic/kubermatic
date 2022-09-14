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

// NewListAKSVersionsParams creates a new ListAKSVersionsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListAKSVersionsParams() *ListAKSVersionsParams {
	return &ListAKSVersionsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListAKSVersionsParamsWithTimeout creates a new ListAKSVersionsParams object
// with the ability to set a timeout on a request.
func NewListAKSVersionsParamsWithTimeout(timeout time.Duration) *ListAKSVersionsParams {
	return &ListAKSVersionsParams{
		timeout: timeout,
	}
}

// NewListAKSVersionsParamsWithContext creates a new ListAKSVersionsParams object
// with the ability to set a context for a request.
func NewListAKSVersionsParamsWithContext(ctx context.Context) *ListAKSVersionsParams {
	return &ListAKSVersionsParams{
		Context: ctx,
	}
}

// NewListAKSVersionsParamsWithHTTPClient creates a new ListAKSVersionsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListAKSVersionsParamsWithHTTPClient(client *http.Client) *ListAKSVersionsParams {
	return &ListAKSVersionsParams{
		HTTPClient: client,
	}
}

/*
ListAKSVersionsParams contains all the parameters to send to the API endpoint

	for the list a k s versions operation.

	Typically these are written to a http.Request.
*/
type ListAKSVersionsParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list a k s versions params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAKSVersionsParams) WithDefaults() *ListAKSVersionsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list a k s versions params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAKSVersionsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list a k s versions params
func (o *ListAKSVersionsParams) WithTimeout(timeout time.Duration) *ListAKSVersionsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list a k s versions params
func (o *ListAKSVersionsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list a k s versions params
func (o *ListAKSVersionsParams) WithContext(ctx context.Context) *ListAKSVersionsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list a k s versions params
func (o *ListAKSVersionsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list a k s versions params
func (o *ListAKSVersionsParams) WithHTTPClient(client *http.Client) *ListAKSVersionsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list a k s versions params
func (o *ListAKSVersionsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListAKSVersionsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
