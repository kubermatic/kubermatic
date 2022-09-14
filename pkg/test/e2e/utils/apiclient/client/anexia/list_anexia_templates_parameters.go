// Code generated by go-swagger; DO NOT EDIT.

package anexia

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

// NewListAnexiaTemplatesParams creates a new ListAnexiaTemplatesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListAnexiaTemplatesParams() *ListAnexiaTemplatesParams {
	return &ListAnexiaTemplatesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListAnexiaTemplatesParamsWithTimeout creates a new ListAnexiaTemplatesParams object
// with the ability to set a timeout on a request.
func NewListAnexiaTemplatesParamsWithTimeout(timeout time.Duration) *ListAnexiaTemplatesParams {
	return &ListAnexiaTemplatesParams{
		timeout: timeout,
	}
}

// NewListAnexiaTemplatesParamsWithContext creates a new ListAnexiaTemplatesParams object
// with the ability to set a context for a request.
func NewListAnexiaTemplatesParamsWithContext(ctx context.Context) *ListAnexiaTemplatesParams {
	return &ListAnexiaTemplatesParams{
		Context: ctx,
	}
}

// NewListAnexiaTemplatesParamsWithHTTPClient creates a new ListAnexiaTemplatesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListAnexiaTemplatesParamsWithHTTPClient(client *http.Client) *ListAnexiaTemplatesParams {
	return &ListAnexiaTemplatesParams{
		HTTPClient: client,
	}
}

/*
ListAnexiaTemplatesParams contains all the parameters to send to the API endpoint

	for the list anexia templates operation.

	Typically these are written to a http.Request.
*/
type ListAnexiaTemplatesParams struct {

	// Credential.
	Credential *string

	// Location.
	Location *string

	// Token.
	Token *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list anexia templates params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAnexiaTemplatesParams) WithDefaults() *ListAnexiaTemplatesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list anexia templates params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAnexiaTemplatesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list anexia templates params
func (o *ListAnexiaTemplatesParams) WithTimeout(timeout time.Duration) *ListAnexiaTemplatesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list anexia templates params
func (o *ListAnexiaTemplatesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list anexia templates params
func (o *ListAnexiaTemplatesParams) WithContext(ctx context.Context) *ListAnexiaTemplatesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list anexia templates params
func (o *ListAnexiaTemplatesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list anexia templates params
func (o *ListAnexiaTemplatesParams) WithHTTPClient(client *http.Client) *ListAnexiaTemplatesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list anexia templates params
func (o *ListAnexiaTemplatesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCredential adds the credential to the list anexia templates params
func (o *ListAnexiaTemplatesParams) WithCredential(credential *string) *ListAnexiaTemplatesParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list anexia templates params
func (o *ListAnexiaTemplatesParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithLocation adds the location to the list anexia templates params
func (o *ListAnexiaTemplatesParams) WithLocation(location *string) *ListAnexiaTemplatesParams {
	o.SetLocation(location)
	return o
}

// SetLocation adds the location to the list anexia templates params
func (o *ListAnexiaTemplatesParams) SetLocation(location *string) {
	o.Location = location
}

// WithToken adds the token to the list anexia templates params
func (o *ListAnexiaTemplatesParams) WithToken(token *string) *ListAnexiaTemplatesParams {
	o.SetToken(token)
	return o
}

// SetToken adds the token to the list anexia templates params
func (o *ListAnexiaTemplatesParams) SetToken(token *string) {
	o.Token = token
}

// WriteToRequest writes these params to a swagger request
func (o *ListAnexiaTemplatesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Credential != nil {

		// header param Credential
		if err := r.SetHeaderParam("Credential", *o.Credential); err != nil {
			return err
		}
	}

	if o.Location != nil {

		// header param Location
		if err := r.SetHeaderParam("Location", *o.Location); err != nil {
			return err
		}
	}

	if o.Token != nil {

		// header param Token
		if err := r.SetHeaderParam("Token", *o.Token); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
