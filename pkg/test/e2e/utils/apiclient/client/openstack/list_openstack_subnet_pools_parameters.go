// Code generated by go-swagger; DO NOT EDIT.

package openstack

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
	"github.com/go-openapi/swag"
)

// NewListOpenstackSubnetPoolsParams creates a new ListOpenstackSubnetPoolsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListOpenstackSubnetPoolsParams() *ListOpenstackSubnetPoolsParams {
	return &ListOpenstackSubnetPoolsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListOpenstackSubnetPoolsParamsWithTimeout creates a new ListOpenstackSubnetPoolsParams object
// with the ability to set a timeout on a request.
func NewListOpenstackSubnetPoolsParamsWithTimeout(timeout time.Duration) *ListOpenstackSubnetPoolsParams {
	return &ListOpenstackSubnetPoolsParams{
		timeout: timeout,
	}
}

// NewListOpenstackSubnetPoolsParamsWithContext creates a new ListOpenstackSubnetPoolsParams object
// with the ability to set a context for a request.
func NewListOpenstackSubnetPoolsParamsWithContext(ctx context.Context) *ListOpenstackSubnetPoolsParams {
	return &ListOpenstackSubnetPoolsParams{
		Context: ctx,
	}
}

// NewListOpenstackSubnetPoolsParamsWithHTTPClient creates a new ListOpenstackSubnetPoolsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListOpenstackSubnetPoolsParamsWithHTTPClient(client *http.Client) *ListOpenstackSubnetPoolsParams {
	return &ListOpenstackSubnetPoolsParams{
		HTTPClient: client,
	}
}

/*
ListOpenstackSubnetPoolsParams contains all the parameters to send to the API endpoint

	for the list openstack subnet pools operation.

	Typically these are written to a http.Request.
*/
type ListOpenstackSubnetPoolsParams struct {

	// ApplicationCredentialID.
	ApplicationCredentialID *string

	// ApplicationCredentialSecret.
	ApplicationCredentialSecret *string

	// Credential.
	Credential *string

	// DatacenterName.
	DatacenterName *string

	// Domain.
	Domain *string

	// OIDCAuthentication.
	OIDCAuthentication *bool

	// Password.
	Password *string

	// Project.
	Project *string

	// ProjectID.
	ProjectID *string

	// Tenant.
	Tenant *string

	// TenantID.
	TenantID *string

	// Username.
	Username *string

	// IPVersion.
	//
	// Format: int64
	IPVersion *int64

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list openstack subnet pools params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListOpenstackSubnetPoolsParams) WithDefaults() *ListOpenstackSubnetPoolsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list openstack subnet pools params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListOpenstackSubnetPoolsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithTimeout(timeout time.Duration) *ListOpenstackSubnetPoolsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithContext(ctx context.Context) *ListOpenstackSubnetPoolsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithHTTPClient(client *http.Client) *ListOpenstackSubnetPoolsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithApplicationCredentialID adds the applicationCredentialID to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithApplicationCredentialID(applicationCredentialID *string) *ListOpenstackSubnetPoolsParams {
	o.SetApplicationCredentialID(applicationCredentialID)
	return o
}

// SetApplicationCredentialID adds the applicationCredentialId to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetApplicationCredentialID(applicationCredentialID *string) {
	o.ApplicationCredentialID = applicationCredentialID
}

// WithApplicationCredentialSecret adds the applicationCredentialSecret to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithApplicationCredentialSecret(applicationCredentialSecret *string) *ListOpenstackSubnetPoolsParams {
	o.SetApplicationCredentialSecret(applicationCredentialSecret)
	return o
}

// SetApplicationCredentialSecret adds the applicationCredentialSecret to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetApplicationCredentialSecret(applicationCredentialSecret *string) {
	o.ApplicationCredentialSecret = applicationCredentialSecret
}

// WithCredential adds the credential to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithCredential(credential *string) *ListOpenstackSubnetPoolsParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithDatacenterName adds the datacenterName to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithDatacenterName(datacenterName *string) *ListOpenstackSubnetPoolsParams {
	o.SetDatacenterName(datacenterName)
	return o
}

// SetDatacenterName adds the datacenterName to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetDatacenterName(datacenterName *string) {
	o.DatacenterName = datacenterName
}

// WithDomain adds the domain to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithDomain(domain *string) *ListOpenstackSubnetPoolsParams {
	o.SetDomain(domain)
	return o
}

// SetDomain adds the domain to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetDomain(domain *string) {
	o.Domain = domain
}

// WithOIDCAuthentication adds the oIDCAuthentication to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithOIDCAuthentication(oIDCAuthentication *bool) *ListOpenstackSubnetPoolsParams {
	o.SetOIDCAuthentication(oIDCAuthentication)
	return o
}

// SetOIDCAuthentication adds the oIdCAuthentication to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetOIDCAuthentication(oIDCAuthentication *bool) {
	o.OIDCAuthentication = oIDCAuthentication
}

// WithPassword adds the password to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithPassword(password *string) *ListOpenstackSubnetPoolsParams {
	o.SetPassword(password)
	return o
}

// SetPassword adds the password to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetPassword(password *string) {
	o.Password = password
}

// WithProject adds the project to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithProject(project *string) *ListOpenstackSubnetPoolsParams {
	o.SetProject(project)
	return o
}

// SetProject adds the project to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetProject(project *string) {
	o.Project = project
}

// WithProjectID adds the projectID to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithProjectID(projectID *string) *ListOpenstackSubnetPoolsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetProjectID(projectID *string) {
	o.ProjectID = projectID
}

// WithTenant adds the tenant to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithTenant(tenant *string) *ListOpenstackSubnetPoolsParams {
	o.SetTenant(tenant)
	return o
}

// SetTenant adds the tenant to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetTenant(tenant *string) {
	o.Tenant = tenant
}

// WithTenantID adds the tenantID to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithTenantID(tenantID *string) *ListOpenstackSubnetPoolsParams {
	o.SetTenantID(tenantID)
	return o
}

// SetTenantID adds the tenantId to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetTenantID(tenantID *string) {
	o.TenantID = tenantID
}

// WithUsername adds the username to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithUsername(username *string) *ListOpenstackSubnetPoolsParams {
	o.SetUsername(username)
	return o
}

// SetUsername adds the username to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetUsername(username *string) {
	o.Username = username
}

// WithIPVersion adds the iPVersion to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) WithIPVersion(iPVersion *int64) *ListOpenstackSubnetPoolsParams {
	o.SetIPVersion(iPVersion)
	return o
}

// SetIPVersion adds the ipVersion to the list openstack subnet pools params
func (o *ListOpenstackSubnetPoolsParams) SetIPVersion(iPVersion *int64) {
	o.IPVersion = iPVersion
}

// WriteToRequest writes these params to a swagger request
func (o *ListOpenstackSubnetPoolsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.ApplicationCredentialID != nil {

		// header param ApplicationCredentialID
		if err := r.SetHeaderParam("ApplicationCredentialID", *o.ApplicationCredentialID); err != nil {
			return err
		}
	}

	if o.ApplicationCredentialSecret != nil {

		// header param ApplicationCredentialSecret
		if err := r.SetHeaderParam("ApplicationCredentialSecret", *o.ApplicationCredentialSecret); err != nil {
			return err
		}
	}

	if o.Credential != nil {

		// header param Credential
		if err := r.SetHeaderParam("Credential", *o.Credential); err != nil {
			return err
		}
	}

	if o.DatacenterName != nil {

		// header param DatacenterName
		if err := r.SetHeaderParam("DatacenterName", *o.DatacenterName); err != nil {
			return err
		}
	}

	if o.Domain != nil {

		// header param Domain
		if err := r.SetHeaderParam("Domain", *o.Domain); err != nil {
			return err
		}
	}

	if o.OIDCAuthentication != nil {

		// header param OIDCAuthentication
		if err := r.SetHeaderParam("OIDCAuthentication", swag.FormatBool(*o.OIDCAuthentication)); err != nil {
			return err
		}
	}

	if o.Password != nil {

		// header param Password
		if err := r.SetHeaderParam("Password", *o.Password); err != nil {
			return err
		}
	}

	if o.Project != nil {

		// header param Project
		if err := r.SetHeaderParam("Project", *o.Project); err != nil {
			return err
		}
	}

	if o.ProjectID != nil {

		// header param ProjectID
		if err := r.SetHeaderParam("ProjectID", *o.ProjectID); err != nil {
			return err
		}
	}

	if o.Tenant != nil {

		// header param Tenant
		if err := r.SetHeaderParam("Tenant", *o.Tenant); err != nil {
			return err
		}
	}

	if o.TenantID != nil {

		// header param TenantID
		if err := r.SetHeaderParam("TenantID", *o.TenantID); err != nil {
			return err
		}
	}

	if o.Username != nil {

		// header param Username
		if err := r.SetHeaderParam("Username", *o.Username); err != nil {
			return err
		}
	}

	if o.IPVersion != nil {

		// query param ip_version
		var qrIPVersion int64

		if o.IPVersion != nil {
			qrIPVersion = *o.IPVersion
		}
		qIPVersion := swag.FormatInt64(qrIPVersion)
		if qIPVersion != "" {

			if err := r.SetQueryParam("ip_version", qIPVersion); err != nil {
				return err
			}
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
