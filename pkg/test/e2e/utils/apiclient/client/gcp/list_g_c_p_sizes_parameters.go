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
	"github.com/go-openapi/strfmt"
)

// NewListGCPSizesParams creates a new ListGCPSizesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListGCPSizesParams() *ListGCPSizesParams {
	return &ListGCPSizesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListGCPSizesParamsWithTimeout creates a new ListGCPSizesParams object
// with the ability to set a timeout on a request.
func NewListGCPSizesParamsWithTimeout(timeout time.Duration) *ListGCPSizesParams {
	return &ListGCPSizesParams{
		timeout: timeout,
	}
}

// NewListGCPSizesParamsWithContext creates a new ListGCPSizesParams object
// with the ability to set a context for a request.
func NewListGCPSizesParamsWithContext(ctx context.Context) *ListGCPSizesParams {
	return &ListGCPSizesParams{
		Context: ctx,
	}
}

// NewListGCPSizesParamsWithHTTPClient creates a new ListGCPSizesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListGCPSizesParamsWithHTTPClient(client *http.Client) *ListGCPSizesParams {
	return &ListGCPSizesParams{
		HTTPClient: client,
	}
}

/* ListGCPSizesParams contains all the parameters to send to the API endpoint
   for the list g c p sizes operation.

   Typically these are written to a http.Request.
*/
type ListGCPSizesParams struct {

	// Credential.
	Credential *string

	// ServiceAccount.
	ServiceAccount *string

	// Zone.
	Zone *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list g c p sizes params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListGCPSizesParams) WithDefaults() *ListGCPSizesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list g c p sizes params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListGCPSizesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list g c p sizes params
func (o *ListGCPSizesParams) WithTimeout(timeout time.Duration) *ListGCPSizesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list g c p sizes params
func (o *ListGCPSizesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list g c p sizes params
func (o *ListGCPSizesParams) WithContext(ctx context.Context) *ListGCPSizesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list g c p sizes params
func (o *ListGCPSizesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list g c p sizes params
func (o *ListGCPSizesParams) WithHTTPClient(client *http.Client) *ListGCPSizesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list g c p sizes params
func (o *ListGCPSizesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCredential adds the credential to the list g c p sizes params
func (o *ListGCPSizesParams) WithCredential(credential *string) *ListGCPSizesParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list g c p sizes params
func (o *ListGCPSizesParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithServiceAccount adds the serviceAccount to the list g c p sizes params
func (o *ListGCPSizesParams) WithServiceAccount(serviceAccount *string) *ListGCPSizesParams {
	o.SetServiceAccount(serviceAccount)
	return o
}

// SetServiceAccount adds the serviceAccount to the list g c p sizes params
func (o *ListGCPSizesParams) SetServiceAccount(serviceAccount *string) {
	o.ServiceAccount = serviceAccount
}

// WithZone adds the zone to the list g c p sizes params
func (o *ListGCPSizesParams) WithZone(zone *string) *ListGCPSizesParams {
	o.SetZone(zone)
	return o
}

// SetZone adds the zone to the list g c p sizes params
func (o *ListGCPSizesParams) SetZone(zone *string) {
	o.Zone = zone
}

// WriteToRequest writes these params to a swagger request
func (o *ListGCPSizesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if o.Zone != nil {

		// header param Zone
		if err := r.SetHeaderParam("Zone", *o.Zone); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
