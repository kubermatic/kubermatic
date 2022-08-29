// Code generated by go-swagger; DO NOT EDIT.

package eks

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

// NewListEKSSecurityGroupsParams creates a new ListEKSSecurityGroupsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListEKSSecurityGroupsParams() *ListEKSSecurityGroupsParams {
	return &ListEKSSecurityGroupsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListEKSSecurityGroupsParamsWithTimeout creates a new ListEKSSecurityGroupsParams object
// with the ability to set a timeout on a request.
func NewListEKSSecurityGroupsParamsWithTimeout(timeout time.Duration) *ListEKSSecurityGroupsParams {
	return &ListEKSSecurityGroupsParams{
		timeout: timeout,
	}
}

// NewListEKSSecurityGroupsParamsWithContext creates a new ListEKSSecurityGroupsParams object
// with the ability to set a context for a request.
func NewListEKSSecurityGroupsParamsWithContext(ctx context.Context) *ListEKSSecurityGroupsParams {
	return &ListEKSSecurityGroupsParams{
		Context: ctx,
	}
}

// NewListEKSSecurityGroupsParamsWithHTTPClient creates a new ListEKSSecurityGroupsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListEKSSecurityGroupsParamsWithHTTPClient(client *http.Client) *ListEKSSecurityGroupsParams {
	return &ListEKSSecurityGroupsParams{
		HTTPClient: client,
	}
}

/*
ListEKSSecurityGroupsParams contains all the parameters to send to the API endpoint

	for the list e k s security groups operation.

	Typically these are written to a http.Request.
*/
type ListEKSSecurityGroupsParams struct {

	// AccessKeyID.
	AccessKeyID *string

	// Credential.
	Credential *string

	// Region.
	Region *string

	// SecretAccessKey.
	SecretAccessKey *string

	// VpcID.
	VpcID *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list e k s security groups params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListEKSSecurityGroupsParams) WithDefaults() *ListEKSSecurityGroupsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list e k s security groups params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListEKSSecurityGroupsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) WithTimeout(timeout time.Duration) *ListEKSSecurityGroupsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) WithContext(ctx context.Context) *ListEKSSecurityGroupsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) WithHTTPClient(client *http.Client) *ListEKSSecurityGroupsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAccessKeyID adds the accessKeyID to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) WithAccessKeyID(accessKeyID *string) *ListEKSSecurityGroupsParams {
	o.SetAccessKeyID(accessKeyID)
	return o
}

// SetAccessKeyID adds the accessKeyId to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) SetAccessKeyID(accessKeyID *string) {
	o.AccessKeyID = accessKeyID
}

// WithCredential adds the credential to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) WithCredential(credential *string) *ListEKSSecurityGroupsParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithRegion adds the region to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) WithRegion(region *string) *ListEKSSecurityGroupsParams {
	o.SetRegion(region)
	return o
}

// SetRegion adds the region to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) SetRegion(region *string) {
	o.Region = region
}

// WithSecretAccessKey adds the secretAccessKey to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) WithSecretAccessKey(secretAccessKey *string) *ListEKSSecurityGroupsParams {
	o.SetSecretAccessKey(secretAccessKey)
	return o
}

// SetSecretAccessKey adds the secretAccessKey to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) SetSecretAccessKey(secretAccessKey *string) {
	o.SecretAccessKey = secretAccessKey
}

// WithVpcID adds the vpcID to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) WithVpcID(vpcID *string) *ListEKSSecurityGroupsParams {
	o.SetVpcID(vpcID)
	return o
}

// SetVpcID adds the vpcId to the list e k s security groups params
func (o *ListEKSSecurityGroupsParams) SetVpcID(vpcID *string) {
	o.VpcID = vpcID
}

// WriteToRequest writes these params to a swagger request
func (o *ListEKSSecurityGroupsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.AccessKeyID != nil {

		// header param AccessKeyID
		if err := r.SetHeaderParam("AccessKeyID", *o.AccessKeyID); err != nil {
			return err
		}
	}

	if o.Credential != nil {

		// header param Credential
		if err := r.SetHeaderParam("Credential", *o.Credential); err != nil {
			return err
		}
	}

	if o.Region != nil {

		// header param Region
		if err := r.SetHeaderParam("Region", *o.Region); err != nil {
			return err
		}
	}

	if o.SecretAccessKey != nil {

		// header param SecretAccessKey
		if err := r.SetHeaderParam("SecretAccessKey", *o.SecretAccessKey); err != nil {
			return err
		}
	}

	if o.VpcID != nil {

		// header param VpcId
		if err := r.SetHeaderParam("VpcId", *o.VpcID); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
