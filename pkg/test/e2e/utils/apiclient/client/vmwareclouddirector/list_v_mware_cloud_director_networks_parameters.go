// Code generated by go-swagger; DO NOT EDIT.

package vmwareclouddirector

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

// NewListVMwareCloudDirectorNetworksParams creates a new ListVMwareCloudDirectorNetworksParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListVMwareCloudDirectorNetworksParams() *ListVMwareCloudDirectorNetworksParams {
	return &ListVMwareCloudDirectorNetworksParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListVMwareCloudDirectorNetworksParamsWithTimeout creates a new ListVMwareCloudDirectorNetworksParams object
// with the ability to set a timeout on a request.
func NewListVMwareCloudDirectorNetworksParamsWithTimeout(timeout time.Duration) *ListVMwareCloudDirectorNetworksParams {
	return &ListVMwareCloudDirectorNetworksParams{
		timeout: timeout,
	}
}

// NewListVMwareCloudDirectorNetworksParamsWithContext creates a new ListVMwareCloudDirectorNetworksParams object
// with the ability to set a context for a request.
func NewListVMwareCloudDirectorNetworksParamsWithContext(ctx context.Context) *ListVMwareCloudDirectorNetworksParams {
	return &ListVMwareCloudDirectorNetworksParams{
		Context: ctx,
	}
}

// NewListVMwareCloudDirectorNetworksParamsWithHTTPClient creates a new ListVMwareCloudDirectorNetworksParams object
// with the ability to set a custom HTTPClient for a request.
func NewListVMwareCloudDirectorNetworksParamsWithHTTPClient(client *http.Client) *ListVMwareCloudDirectorNetworksParams {
	return &ListVMwareCloudDirectorNetworksParams{
		HTTPClient: client,
	}
}

/*
ListVMwareCloudDirectorNetworksParams contains all the parameters to send to the API endpoint

	for the list v mware cloud director networks operation.

	Typically these are written to a http.Request.
*/
type ListVMwareCloudDirectorNetworksParams struct {

	// Credential.
	Credential *string

	// Organization.
	Organization *string

	// Password.
	Password *string

	// Username.
	Username *string

	// VDC.
	VDC *string

	/* Dc.

	   KKP Datacenter to use for endpoint
	*/
	DC string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list v mware cloud director networks params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListVMwareCloudDirectorNetworksParams) WithDefaults() *ListVMwareCloudDirectorNetworksParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list v mware cloud director networks params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListVMwareCloudDirectorNetworksParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) WithTimeout(timeout time.Duration) *ListVMwareCloudDirectorNetworksParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) WithContext(ctx context.Context) *ListVMwareCloudDirectorNetworksParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) WithHTTPClient(client *http.Client) *ListVMwareCloudDirectorNetworksParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCredential adds the credential to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) WithCredential(credential *string) *ListVMwareCloudDirectorNetworksParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithOrganization adds the organization to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) WithOrganization(organization *string) *ListVMwareCloudDirectorNetworksParams {
	o.SetOrganization(organization)
	return o
}

// SetOrganization adds the organization to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) SetOrganization(organization *string) {
	o.Organization = organization
}

// WithPassword adds the password to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) WithPassword(password *string) *ListVMwareCloudDirectorNetworksParams {
	o.SetPassword(password)
	return o
}

// SetPassword adds the password to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) SetPassword(password *string) {
	o.Password = password
}

// WithUsername adds the username to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) WithUsername(username *string) *ListVMwareCloudDirectorNetworksParams {
	o.SetUsername(username)
	return o
}

// SetUsername adds the username to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) SetUsername(username *string) {
	o.Username = username
}

// WithVDC adds the vDC to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) WithVDC(vDC *string) *ListVMwareCloudDirectorNetworksParams {
	o.SetVDC(vDC)
	return o
}

// SetVDC adds the vDC to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) SetVDC(vDC *string) {
	o.VDC = vDC
}

// WithDC adds the dc to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) WithDC(dc string) *ListVMwareCloudDirectorNetworksParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list v mware cloud director networks params
func (o *ListVMwareCloudDirectorNetworksParams) SetDC(dc string) {
	o.DC = dc
}

// WriteToRequest writes these params to a swagger request
func (o *ListVMwareCloudDirectorNetworksParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if o.Organization != nil {

		// header param Organization
		if err := r.SetHeaderParam("Organization", *o.Organization); err != nil {
			return err
		}
	}

	if o.Password != nil {

		// header param Password
		if err := r.SetHeaderParam("Password", *o.Password); err != nil {
			return err
		}
	}

	if o.Username != nil {

		// header param Username
		if err := r.SetHeaderParam("Username", *o.Username); err != nil {
			return err
		}
	}

	if o.VDC != nil {

		// header param VDC
		if err := r.SetHeaderParam("VDC", *o.VDC); err != nil {
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
