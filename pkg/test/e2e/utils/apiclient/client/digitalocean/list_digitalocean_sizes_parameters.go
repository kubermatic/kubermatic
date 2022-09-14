// Code generated by go-swagger; DO NOT EDIT.

package digitalocean

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

// NewListDigitaloceanSizesParams creates a new ListDigitaloceanSizesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListDigitaloceanSizesParams() *ListDigitaloceanSizesParams {
	return &ListDigitaloceanSizesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListDigitaloceanSizesParamsWithTimeout creates a new ListDigitaloceanSizesParams object
// with the ability to set a timeout on a request.
func NewListDigitaloceanSizesParamsWithTimeout(timeout time.Duration) *ListDigitaloceanSizesParams {
	return &ListDigitaloceanSizesParams{
		timeout: timeout,
	}
}

// NewListDigitaloceanSizesParamsWithContext creates a new ListDigitaloceanSizesParams object
// with the ability to set a context for a request.
func NewListDigitaloceanSizesParamsWithContext(ctx context.Context) *ListDigitaloceanSizesParams {
	return &ListDigitaloceanSizesParams{
		Context: ctx,
	}
}

// NewListDigitaloceanSizesParamsWithHTTPClient creates a new ListDigitaloceanSizesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListDigitaloceanSizesParamsWithHTTPClient(client *http.Client) *ListDigitaloceanSizesParams {
	return &ListDigitaloceanSizesParams{
		HTTPClient: client,
	}
}

/*
ListDigitaloceanSizesParams contains all the parameters to send to the API endpoint

	for the list digitalocean sizes operation.

	Typically these are written to a http.Request.
*/
type ListDigitaloceanSizesParams struct {

	// Credential.
	Credential *string

	// DoToken.
	DoToken *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list digitalocean sizes params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListDigitaloceanSizesParams) WithDefaults() *ListDigitaloceanSizesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list digitalocean sizes params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListDigitaloceanSizesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list digitalocean sizes params
func (o *ListDigitaloceanSizesParams) WithTimeout(timeout time.Duration) *ListDigitaloceanSizesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list digitalocean sizes params
func (o *ListDigitaloceanSizesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list digitalocean sizes params
func (o *ListDigitaloceanSizesParams) WithContext(ctx context.Context) *ListDigitaloceanSizesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list digitalocean sizes params
func (o *ListDigitaloceanSizesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list digitalocean sizes params
func (o *ListDigitaloceanSizesParams) WithHTTPClient(client *http.Client) *ListDigitaloceanSizesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list digitalocean sizes params
func (o *ListDigitaloceanSizesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCredential adds the credential to the list digitalocean sizes params
func (o *ListDigitaloceanSizesParams) WithCredential(credential *string) *ListDigitaloceanSizesParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list digitalocean sizes params
func (o *ListDigitaloceanSizesParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithDoToken adds the doToken to the list digitalocean sizes params
func (o *ListDigitaloceanSizesParams) WithDoToken(doToken *string) *ListDigitaloceanSizesParams {
	o.SetDoToken(doToken)
	return o
}

// SetDoToken adds the doToken to the list digitalocean sizes params
func (o *ListDigitaloceanSizesParams) SetDoToken(doToken *string) {
	o.DoToken = doToken
}

// WriteToRequest writes these params to a swagger request
func (o *ListDigitaloceanSizesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if o.DoToken != nil {

		// header param DoToken
		if err := r.SetHeaderParam("DoToken", *o.DoToken); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
