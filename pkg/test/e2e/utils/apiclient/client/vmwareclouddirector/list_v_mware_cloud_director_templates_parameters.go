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

// NewListVMwareCloudDirectorTemplatesParams creates a new ListVMwareCloudDirectorTemplatesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListVMwareCloudDirectorTemplatesParams() *ListVMwareCloudDirectorTemplatesParams {
	return &ListVMwareCloudDirectorTemplatesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListVMwareCloudDirectorTemplatesParamsWithTimeout creates a new ListVMwareCloudDirectorTemplatesParams object
// with the ability to set a timeout on a request.
func NewListVMwareCloudDirectorTemplatesParamsWithTimeout(timeout time.Duration) *ListVMwareCloudDirectorTemplatesParams {
	return &ListVMwareCloudDirectorTemplatesParams{
		timeout: timeout,
	}
}

// NewListVMwareCloudDirectorTemplatesParamsWithContext creates a new ListVMwareCloudDirectorTemplatesParams object
// with the ability to set a context for a request.
func NewListVMwareCloudDirectorTemplatesParamsWithContext(ctx context.Context) *ListVMwareCloudDirectorTemplatesParams {
	return &ListVMwareCloudDirectorTemplatesParams{
		Context: ctx,
	}
}

// NewListVMwareCloudDirectorTemplatesParamsWithHTTPClient creates a new ListVMwareCloudDirectorTemplatesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListVMwareCloudDirectorTemplatesParamsWithHTTPClient(client *http.Client) *ListVMwareCloudDirectorTemplatesParams {
	return &ListVMwareCloudDirectorTemplatesParams{
		HTTPClient: client,
	}
}

/*
ListVMwareCloudDirectorTemplatesParams contains all the parameters to send to the API endpoint

	for the list v mware cloud director templates operation.

	Typically these are written to a http.Request.
*/
type ListVMwareCloudDirectorTemplatesParams struct {

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

	/* CatalogName.

	   Catalog name to fetch the templates from
	*/
	CatalogName string

	/* Dc.

	   KKP Datacenter to use for endpoint
	*/
	DC string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list v mware cloud director templates params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListVMwareCloudDirectorTemplatesParams) WithDefaults() *ListVMwareCloudDirectorTemplatesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list v mware cloud director templates params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListVMwareCloudDirectorTemplatesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) WithTimeout(timeout time.Duration) *ListVMwareCloudDirectorTemplatesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) WithContext(ctx context.Context) *ListVMwareCloudDirectorTemplatesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) WithHTTPClient(client *http.Client) *ListVMwareCloudDirectorTemplatesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCredential adds the credential to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) WithCredential(credential *string) *ListVMwareCloudDirectorTemplatesParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithOrganization adds the organization to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) WithOrganization(organization *string) *ListVMwareCloudDirectorTemplatesParams {
	o.SetOrganization(organization)
	return o
}

// SetOrganization adds the organization to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) SetOrganization(organization *string) {
	o.Organization = organization
}

// WithPassword adds the password to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) WithPassword(password *string) *ListVMwareCloudDirectorTemplatesParams {
	o.SetPassword(password)
	return o
}

// SetPassword adds the password to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) SetPassword(password *string) {
	o.Password = password
}

// WithUsername adds the username to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) WithUsername(username *string) *ListVMwareCloudDirectorTemplatesParams {
	o.SetUsername(username)
	return o
}

// SetUsername adds the username to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) SetUsername(username *string) {
	o.Username = username
}

// WithVDC adds the vDC to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) WithVDC(vDC *string) *ListVMwareCloudDirectorTemplatesParams {
	o.SetVDC(vDC)
	return o
}

// SetVDC adds the vDC to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) SetVDC(vDC *string) {
	o.VDC = vDC
}

// WithCatalogName adds the catalogName to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) WithCatalogName(catalogName string) *ListVMwareCloudDirectorTemplatesParams {
	o.SetCatalogName(catalogName)
	return o
}

// SetCatalogName adds the catalogName to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) SetCatalogName(catalogName string) {
	o.CatalogName = catalogName
}

// WithDC adds the dc to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) WithDC(dc string) *ListVMwareCloudDirectorTemplatesParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list v mware cloud director templates params
func (o *ListVMwareCloudDirectorTemplatesParams) SetDC(dc string) {
	o.DC = dc
}

// WriteToRequest writes these params to a swagger request
func (o *ListVMwareCloudDirectorTemplatesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	// path param catalog_name
	if err := r.SetPathParam("catalog_name", o.CatalogName); err != nil {
		return err
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
