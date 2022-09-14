// Code generated by go-swagger; DO NOT EDIT.

package nutanix

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

// NewListNutanixClustersParams creates a new ListNutanixClustersParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListNutanixClustersParams() *ListNutanixClustersParams {
	return &ListNutanixClustersParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListNutanixClustersParamsWithTimeout creates a new ListNutanixClustersParams object
// with the ability to set a timeout on a request.
func NewListNutanixClustersParamsWithTimeout(timeout time.Duration) *ListNutanixClustersParams {
	return &ListNutanixClustersParams{
		timeout: timeout,
	}
}

// NewListNutanixClustersParamsWithContext creates a new ListNutanixClustersParams object
// with the ability to set a context for a request.
func NewListNutanixClustersParamsWithContext(ctx context.Context) *ListNutanixClustersParams {
	return &ListNutanixClustersParams{
		Context: ctx,
	}
}

// NewListNutanixClustersParamsWithHTTPClient creates a new ListNutanixClustersParams object
// with the ability to set a custom HTTPClient for a request.
func NewListNutanixClustersParamsWithHTTPClient(client *http.Client) *ListNutanixClustersParams {
	return &ListNutanixClustersParams{
		HTTPClient: client,
	}
}

/*
ListNutanixClustersParams contains all the parameters to send to the API endpoint

	for the list nutanix clusters operation.

	Typically these are written to a http.Request.
*/
type ListNutanixClustersParams struct {

	// Credential.
	Credential *string

	// NutanixPassword.
	NutanixPassword *string

	// NutanixProxyURL.
	NutanixProxyURL *string

	// NutanixUsername.
	NutanixUsername *string

	/* Dc.

	   KKP Datacenter to use for endpoint
	*/
	DC string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list nutanix clusters params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListNutanixClustersParams) WithDefaults() *ListNutanixClustersParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list nutanix clusters params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListNutanixClustersParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list nutanix clusters params
func (o *ListNutanixClustersParams) WithTimeout(timeout time.Duration) *ListNutanixClustersParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list nutanix clusters params
func (o *ListNutanixClustersParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list nutanix clusters params
func (o *ListNutanixClustersParams) WithContext(ctx context.Context) *ListNutanixClustersParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list nutanix clusters params
func (o *ListNutanixClustersParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list nutanix clusters params
func (o *ListNutanixClustersParams) WithHTTPClient(client *http.Client) *ListNutanixClustersParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list nutanix clusters params
func (o *ListNutanixClustersParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCredential adds the credential to the list nutanix clusters params
func (o *ListNutanixClustersParams) WithCredential(credential *string) *ListNutanixClustersParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list nutanix clusters params
func (o *ListNutanixClustersParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithNutanixPassword adds the nutanixPassword to the list nutanix clusters params
func (o *ListNutanixClustersParams) WithNutanixPassword(nutanixPassword *string) *ListNutanixClustersParams {
	o.SetNutanixPassword(nutanixPassword)
	return o
}

// SetNutanixPassword adds the nutanixPassword to the list nutanix clusters params
func (o *ListNutanixClustersParams) SetNutanixPassword(nutanixPassword *string) {
	o.NutanixPassword = nutanixPassword
}

// WithNutanixProxyURL adds the nutanixProxyURL to the list nutanix clusters params
func (o *ListNutanixClustersParams) WithNutanixProxyURL(nutanixProxyURL *string) *ListNutanixClustersParams {
	o.SetNutanixProxyURL(nutanixProxyURL)
	return o
}

// SetNutanixProxyURL adds the nutanixProxyUrl to the list nutanix clusters params
func (o *ListNutanixClustersParams) SetNutanixProxyURL(nutanixProxyURL *string) {
	o.NutanixProxyURL = nutanixProxyURL
}

// WithNutanixUsername adds the nutanixUsername to the list nutanix clusters params
func (o *ListNutanixClustersParams) WithNutanixUsername(nutanixUsername *string) *ListNutanixClustersParams {
	o.SetNutanixUsername(nutanixUsername)
	return o
}

// SetNutanixUsername adds the nutanixUsername to the list nutanix clusters params
func (o *ListNutanixClustersParams) SetNutanixUsername(nutanixUsername *string) {
	o.NutanixUsername = nutanixUsername
}

// WithDC adds the dc to the list nutanix clusters params
func (o *ListNutanixClustersParams) WithDC(dc string) *ListNutanixClustersParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list nutanix clusters params
func (o *ListNutanixClustersParams) SetDC(dc string) {
	o.DC = dc
}

// WriteToRequest writes these params to a swagger request
func (o *ListNutanixClustersParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if o.NutanixPassword != nil {

		// header param NutanixPassword
		if err := r.SetHeaderParam("NutanixPassword", *o.NutanixPassword); err != nil {
			return err
		}
	}

	if o.NutanixProxyURL != nil {

		// header param NutanixProxyURL
		if err := r.SetHeaderParam("NutanixProxyURL", *o.NutanixProxyURL); err != nil {
			return err
		}
	}

	if o.NutanixUsername != nil {

		// header param NutanixUsername
		if err := r.SetHeaderParam("NutanixUsername", *o.NutanixUsername); err != nil {
			return err
		}
	}

	// path param dc
	if err := r.SetPathParam("dc", o.DC); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
