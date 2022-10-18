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

// NewListVMwareCloudDirectorStorageProfilesParams creates a new ListVMwareCloudDirectorStorageProfilesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListVMwareCloudDirectorStorageProfilesParams() *ListVMwareCloudDirectorStorageProfilesParams {
	return &ListVMwareCloudDirectorStorageProfilesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListVMwareCloudDirectorStorageProfilesParamsWithTimeout creates a new ListVMwareCloudDirectorStorageProfilesParams object
// with the ability to set a timeout on a request.
func NewListVMwareCloudDirectorStorageProfilesParamsWithTimeout(timeout time.Duration) *ListVMwareCloudDirectorStorageProfilesParams {
	return &ListVMwareCloudDirectorStorageProfilesParams{
		timeout: timeout,
	}
}

// NewListVMwareCloudDirectorStorageProfilesParamsWithContext creates a new ListVMwareCloudDirectorStorageProfilesParams object
// with the ability to set a context for a request.
func NewListVMwareCloudDirectorStorageProfilesParamsWithContext(ctx context.Context) *ListVMwareCloudDirectorStorageProfilesParams {
	return &ListVMwareCloudDirectorStorageProfilesParams{
		Context: ctx,
	}
}

// NewListVMwareCloudDirectorStorageProfilesParamsWithHTTPClient creates a new ListVMwareCloudDirectorStorageProfilesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListVMwareCloudDirectorStorageProfilesParamsWithHTTPClient(client *http.Client) *ListVMwareCloudDirectorStorageProfilesParams {
	return &ListVMwareCloudDirectorStorageProfilesParams{
		HTTPClient: client,
	}
}

/*
ListVMwareCloudDirectorStorageProfilesParams contains all the parameters to send to the API endpoint

	for the list v mware cloud director storage profiles operation.

	Typically these are written to a http.Request.
*/
type ListVMwareCloudDirectorStorageProfilesParams struct {

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

// WithDefaults hydrates default values in the list v mware cloud director storage profiles params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListVMwareCloudDirectorStorageProfilesParams) WithDefaults() *ListVMwareCloudDirectorStorageProfilesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list v mware cloud director storage profiles params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListVMwareCloudDirectorStorageProfilesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) WithTimeout(timeout time.Duration) *ListVMwareCloudDirectorStorageProfilesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) WithContext(ctx context.Context) *ListVMwareCloudDirectorStorageProfilesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) WithHTTPClient(client *http.Client) *ListVMwareCloudDirectorStorageProfilesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCredential adds the credential to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) WithCredential(credential *string) *ListVMwareCloudDirectorStorageProfilesParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithOrganization adds the organization to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) WithOrganization(organization *string) *ListVMwareCloudDirectorStorageProfilesParams {
	o.SetOrganization(organization)
	return o
}

// SetOrganization adds the organization to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) SetOrganization(organization *string) {
	o.Organization = organization
}

// WithPassword adds the password to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) WithPassword(password *string) *ListVMwareCloudDirectorStorageProfilesParams {
	o.SetPassword(password)
	return o
}

// SetPassword adds the password to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) SetPassword(password *string) {
	o.Password = password
}

// WithUsername adds the username to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) WithUsername(username *string) *ListVMwareCloudDirectorStorageProfilesParams {
	o.SetUsername(username)
	return o
}

// SetUsername adds the username to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) SetUsername(username *string) {
	o.Username = username
}

// WithVDC adds the vDC to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) WithVDC(vDC *string) *ListVMwareCloudDirectorStorageProfilesParams {
	o.SetVDC(vDC)
	return o
}

// SetVDC adds the vDC to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) SetVDC(vDC *string) {
	o.VDC = vDC
}

// WithDC adds the dc to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) WithDC(dc string) *ListVMwareCloudDirectorStorageProfilesParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list v mware cloud director storage profiles params
func (o *ListVMwareCloudDirectorStorageProfilesParams) SetDC(dc string) {
	o.DC = dc
}

// WriteToRequest writes these params to a swagger request
func (o *ListVMwareCloudDirectorStorageProfilesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
