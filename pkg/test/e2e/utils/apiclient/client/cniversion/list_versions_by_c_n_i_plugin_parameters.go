// Code generated by go-swagger; DO NOT EDIT.

package cniversion

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

// NewListVersionsByCNIPluginParams creates a new ListVersionsByCNIPluginParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListVersionsByCNIPluginParams() *ListVersionsByCNIPluginParams {
	return &ListVersionsByCNIPluginParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListVersionsByCNIPluginParamsWithTimeout creates a new ListVersionsByCNIPluginParams object
// with the ability to set a timeout on a request.
func NewListVersionsByCNIPluginParamsWithTimeout(timeout time.Duration) *ListVersionsByCNIPluginParams {
	return &ListVersionsByCNIPluginParams{
		timeout: timeout,
	}
}

// NewListVersionsByCNIPluginParamsWithContext creates a new ListVersionsByCNIPluginParams object
// with the ability to set a context for a request.
func NewListVersionsByCNIPluginParamsWithContext(ctx context.Context) *ListVersionsByCNIPluginParams {
	return &ListVersionsByCNIPluginParams{
		Context: ctx,
	}
}

// NewListVersionsByCNIPluginParamsWithHTTPClient creates a new ListVersionsByCNIPluginParams object
// with the ability to set a custom HTTPClient for a request.
func NewListVersionsByCNIPluginParamsWithHTTPClient(client *http.Client) *ListVersionsByCNIPluginParams {
	return &ListVersionsByCNIPluginParams{
		HTTPClient: client,
	}
}

/*
ListVersionsByCNIPluginParams contains all the parameters to send to the API endpoint

	for the list versions by c n i plugin operation.

	Typically these are written to a http.Request.
*/
type ListVersionsByCNIPluginParams struct {

	// CniPluginType.
	CNIPluginType string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list versions by c n i plugin params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListVersionsByCNIPluginParams) WithDefaults() *ListVersionsByCNIPluginParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list versions by c n i plugin params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListVersionsByCNIPluginParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list versions by c n i plugin params
func (o *ListVersionsByCNIPluginParams) WithTimeout(timeout time.Duration) *ListVersionsByCNIPluginParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list versions by c n i plugin params
func (o *ListVersionsByCNIPluginParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list versions by c n i plugin params
func (o *ListVersionsByCNIPluginParams) WithContext(ctx context.Context) *ListVersionsByCNIPluginParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list versions by c n i plugin params
func (o *ListVersionsByCNIPluginParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list versions by c n i plugin params
func (o *ListVersionsByCNIPluginParams) WithHTTPClient(client *http.Client) *ListVersionsByCNIPluginParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list versions by c n i plugin params
func (o *ListVersionsByCNIPluginParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCNIPluginType adds the cniPluginType to the list versions by c n i plugin params
func (o *ListVersionsByCNIPluginParams) WithCNIPluginType(cniPluginType string) *ListVersionsByCNIPluginParams {
	o.SetCNIPluginType(cniPluginType)
	return o
}

// SetCNIPluginType adds the cniPluginType to the list versions by c n i plugin params
func (o *ListVersionsByCNIPluginParams) SetCNIPluginType(cniPluginType string) {
	o.CNIPluginType = cniPluginType
}

// WriteToRequest writes these params to a swagger request
func (o *ListVersionsByCNIPluginParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cni_plugin_type
	if err := r.SetPathParam("cni_plugin_type", o.CNIPluginType); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
