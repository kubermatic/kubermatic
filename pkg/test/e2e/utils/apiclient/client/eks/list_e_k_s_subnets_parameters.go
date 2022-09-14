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

// NewListEKSSubnetsParams creates a new ListEKSSubnetsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListEKSSubnetsParams() *ListEKSSubnetsParams {
	return &ListEKSSubnetsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListEKSSubnetsParamsWithTimeout creates a new ListEKSSubnetsParams object
// with the ability to set a timeout on a request.
func NewListEKSSubnetsParamsWithTimeout(timeout time.Duration) *ListEKSSubnetsParams {
	return &ListEKSSubnetsParams{
		timeout: timeout,
	}
}

// NewListEKSSubnetsParamsWithContext creates a new ListEKSSubnetsParams object
// with the ability to set a context for a request.
func NewListEKSSubnetsParamsWithContext(ctx context.Context) *ListEKSSubnetsParams {
	return &ListEKSSubnetsParams{
		Context: ctx,
	}
}

// NewListEKSSubnetsParamsWithHTTPClient creates a new ListEKSSubnetsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListEKSSubnetsParamsWithHTTPClient(client *http.Client) *ListEKSSubnetsParams {
	return &ListEKSSubnetsParams{
		HTTPClient: client,
	}
}

/*
ListEKSSubnetsParams contains all the parameters to send to the API endpoint

	for the list e k s subnets operation.

	Typically these are written to a http.Request.
*/
type ListEKSSubnetsParams struct {

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

// WithDefaults hydrates default values in the list e k s subnets params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListEKSSubnetsParams) WithDefaults() *ListEKSSubnetsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list e k s subnets params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListEKSSubnetsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list e k s subnets params
func (o *ListEKSSubnetsParams) WithTimeout(timeout time.Duration) *ListEKSSubnetsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list e k s subnets params
func (o *ListEKSSubnetsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list e k s subnets params
func (o *ListEKSSubnetsParams) WithContext(ctx context.Context) *ListEKSSubnetsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list e k s subnets params
func (o *ListEKSSubnetsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list e k s subnets params
func (o *ListEKSSubnetsParams) WithHTTPClient(client *http.Client) *ListEKSSubnetsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list e k s subnets params
func (o *ListEKSSubnetsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAccessKeyID adds the accessKeyID to the list e k s subnets params
func (o *ListEKSSubnetsParams) WithAccessKeyID(accessKeyID *string) *ListEKSSubnetsParams {
	o.SetAccessKeyID(accessKeyID)
	return o
}

// SetAccessKeyID adds the accessKeyId to the list e k s subnets params
func (o *ListEKSSubnetsParams) SetAccessKeyID(accessKeyID *string) {
	o.AccessKeyID = accessKeyID
}

// WithCredential adds the credential to the list e k s subnets params
func (o *ListEKSSubnetsParams) WithCredential(credential *string) *ListEKSSubnetsParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list e k s subnets params
func (o *ListEKSSubnetsParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithRegion adds the region to the list e k s subnets params
func (o *ListEKSSubnetsParams) WithRegion(region *string) *ListEKSSubnetsParams {
	o.SetRegion(region)
	return o
}

// SetRegion adds the region to the list e k s subnets params
func (o *ListEKSSubnetsParams) SetRegion(region *string) {
	o.Region = region
}

// WithSecretAccessKey adds the secretAccessKey to the list e k s subnets params
func (o *ListEKSSubnetsParams) WithSecretAccessKey(secretAccessKey *string) *ListEKSSubnetsParams {
	o.SetSecretAccessKey(secretAccessKey)
	return o
}

// SetSecretAccessKey adds the secretAccessKey to the list e k s subnets params
func (o *ListEKSSubnetsParams) SetSecretAccessKey(secretAccessKey *string) {
	o.SecretAccessKey = secretAccessKey
}

// WithVpcID adds the vpcID to the list e k s subnets params
func (o *ListEKSSubnetsParams) WithVpcID(vpcID *string) *ListEKSSubnetsParams {
	o.SetVpcID(vpcID)
	return o
}

// SetVpcID adds the vpcId to the list e k s subnets params
func (o *ListEKSSubnetsParams) SetVpcID(vpcID *string) {
	o.VpcID = vpcID
}

// WriteToRequest writes these params to a swagger request
func (o *ListEKSSubnetsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
