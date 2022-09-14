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
)

// NewListAllowedRegistriesParams creates a new ListAllowedRegistriesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListAllowedRegistriesParams() *ListAllowedRegistriesParams {
	return &ListAllowedRegistriesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListAllowedRegistriesParamsWithTimeout creates a new ListAllowedRegistriesParams object
// with the ability to set a timeout on a request.
func NewListAllowedRegistriesParamsWithTimeout(timeout time.Duration) *ListAllowedRegistriesParams {
	return &ListAllowedRegistriesParams{
		timeout: timeout,
	}
}

// NewListAllowedRegistriesParamsWithContext creates a new ListAllowedRegistriesParams object
// with the ability to set a context for a request.
func NewListAllowedRegistriesParamsWithContext(ctx context.Context) *ListAllowedRegistriesParams {
	return &ListAllowedRegistriesParams{
		Context: ctx,
	}
}

// NewListAllowedRegistriesParamsWithHTTPClient creates a new ListAllowedRegistriesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListAllowedRegistriesParamsWithHTTPClient(client *http.Client) *ListAllowedRegistriesParams {
	return &ListAllowedRegistriesParams{
		HTTPClient: client,
	}
}

/*
ListAllowedRegistriesParams contains all the parameters to send to the API endpoint

	for the list allowed registries operation.

	Typically these are written to a http.Request.
*/
type ListAllowedRegistriesParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list allowed registries params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAllowedRegistriesParams) WithDefaults() *ListAllowedRegistriesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list allowed registries params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAllowedRegistriesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list allowed registries params
func (o *ListAllowedRegistriesParams) WithTimeout(timeout time.Duration) *ListAllowedRegistriesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list allowed registries params
func (o *ListAllowedRegistriesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list allowed registries params
func (o *ListAllowedRegistriesParams) WithContext(ctx context.Context) *ListAllowedRegistriesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list allowed registries params
func (o *ListAllowedRegistriesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list allowed registries params
func (o *ListAllowedRegistriesParams) WithHTTPClient(client *http.Client) *ListAllowedRegistriesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list allowed registries params
func (o *ListAllowedRegistriesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListAllowedRegistriesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
