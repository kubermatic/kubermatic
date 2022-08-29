// Code generated by go-swagger; DO NOT EDIT.

package admin

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

// NewDeleteAdmissionPluginParams creates a new DeleteAdmissionPluginParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteAdmissionPluginParams() *DeleteAdmissionPluginParams {
	return &DeleteAdmissionPluginParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteAdmissionPluginParamsWithTimeout creates a new DeleteAdmissionPluginParams object
// with the ability to set a timeout on a request.
func NewDeleteAdmissionPluginParamsWithTimeout(timeout time.Duration) *DeleteAdmissionPluginParams {
	return &DeleteAdmissionPluginParams{
		timeout: timeout,
	}
}

// NewDeleteAdmissionPluginParamsWithContext creates a new DeleteAdmissionPluginParams object
// with the ability to set a context for a request.
func NewDeleteAdmissionPluginParamsWithContext(ctx context.Context) *DeleteAdmissionPluginParams {
	return &DeleteAdmissionPluginParams{
		Context: ctx,
	}
}

// NewDeleteAdmissionPluginParamsWithHTTPClient creates a new DeleteAdmissionPluginParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteAdmissionPluginParamsWithHTTPClient(client *http.Client) *DeleteAdmissionPluginParams {
	return &DeleteAdmissionPluginParams{
		HTTPClient: client,
	}
}

/*
DeleteAdmissionPluginParams contains all the parameters to send to the API endpoint

	for the delete admission plugin operation.

	Typically these are written to a http.Request.
*/
type DeleteAdmissionPluginParams struct {

	// Name.
	Name string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete admission plugin params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteAdmissionPluginParams) WithDefaults() *DeleteAdmissionPluginParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete admission plugin params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteAdmissionPluginParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete admission plugin params
func (o *DeleteAdmissionPluginParams) WithTimeout(timeout time.Duration) *DeleteAdmissionPluginParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete admission plugin params
func (o *DeleteAdmissionPluginParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete admission plugin params
func (o *DeleteAdmissionPluginParams) WithContext(ctx context.Context) *DeleteAdmissionPluginParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete admission plugin params
func (o *DeleteAdmissionPluginParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete admission plugin params
func (o *DeleteAdmissionPluginParams) WithHTTPClient(client *http.Client) *DeleteAdmissionPluginParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete admission plugin params
func (o *DeleteAdmissionPluginParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithName adds the name to the delete admission plugin params
func (o *DeleteAdmissionPluginParams) WithName(name string) *DeleteAdmissionPluginParams {
	o.SetName(name)
	return o
}

// SetName adds the name to the delete admission plugin params
func (o *DeleteAdmissionPluginParams) SetName(name string) {
	o.Name = name
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteAdmissionPluginParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param name
	if err := r.SetPathParam("name", o.Name); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
