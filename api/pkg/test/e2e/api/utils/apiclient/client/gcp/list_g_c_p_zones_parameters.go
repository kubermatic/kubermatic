// Code generated by go-swagger; DO NOT EDIT.

package gcp

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"

	strfmt "github.com/go-openapi/strfmt"
)

// NewListGCPZonesParams creates a new ListGCPZonesParams object
// with the default values initialized.
func NewListGCPZonesParams() *ListGCPZonesParams {
	var ()
	return &ListGCPZonesParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewListGCPZonesParamsWithTimeout creates a new ListGCPZonesParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListGCPZonesParamsWithTimeout(timeout time.Duration) *ListGCPZonesParams {
	var ()
	return &ListGCPZonesParams{

		timeout: timeout,
	}
}

// NewListGCPZonesParamsWithContext creates a new ListGCPZonesParams object
// with the default values initialized, and the ability to set a context for a request
func NewListGCPZonesParamsWithContext(ctx context.Context) *ListGCPZonesParams {
	var ()
	return &ListGCPZonesParams{

		Context: ctx,
	}
}

// NewListGCPZonesParamsWithHTTPClient creates a new ListGCPZonesParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListGCPZonesParamsWithHTTPClient(client *http.Client) *ListGCPZonesParams {
	var ()
	return &ListGCPZonesParams{
		HTTPClient: client,
	}
}

/*ListGCPZonesParams contains all the parameters to send to the API endpoint
for the list g c p zones operation typically these are written to a http.Request
*/
type ListGCPZonesParams struct {

	/*Credential*/
	Credential *string
	/*ServiceAccount*/
	ServiceAccount *string
	/*Dc*/
	Dc string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list g c p zones params
func (o *ListGCPZonesParams) WithTimeout(timeout time.Duration) *ListGCPZonesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list g c p zones params
func (o *ListGCPZonesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list g c p zones params
func (o *ListGCPZonesParams) WithContext(ctx context.Context) *ListGCPZonesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list g c p zones params
func (o *ListGCPZonesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list g c p zones params
func (o *ListGCPZonesParams) WithHTTPClient(client *http.Client) *ListGCPZonesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list g c p zones params
func (o *ListGCPZonesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCredential adds the credential to the list g c p zones params
func (o *ListGCPZonesParams) WithCredential(credential *string) *ListGCPZonesParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list g c p zones params
func (o *ListGCPZonesParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithServiceAccount adds the serviceAccount to the list g c p zones params
func (o *ListGCPZonesParams) WithServiceAccount(serviceAccount *string) *ListGCPZonesParams {
	o.SetServiceAccount(serviceAccount)
	return o
}

// SetServiceAccount adds the serviceAccount to the list g c p zones params
func (o *ListGCPZonesParams) SetServiceAccount(serviceAccount *string) {
	o.ServiceAccount = serviceAccount
}

// WithDc adds the dc to the list g c p zones params
func (o *ListGCPZonesParams) WithDc(dc string) *ListGCPZonesParams {
	o.SetDc(dc)
	return o
}

// SetDc adds the dc to the list g c p zones params
func (o *ListGCPZonesParams) SetDc(dc string) {
	o.Dc = dc
}

// WriteToRequest writes these params to a swagger request
func (o *ListGCPZonesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if o.ServiceAccount != nil {

		// header param ServiceAccount
		if err := r.SetHeaderParam("ServiceAccount", *o.ServiceAccount); err != nil {
			return err
		}

	}

	// path param dc
	if err := r.SetPathParam("dc", o.Dc); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
