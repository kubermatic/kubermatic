// Code generated by go-swagger; DO NOT EDIT.

package aws

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

// NewListAWSSubnetsParams creates a new ListAWSSubnetsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListAWSSubnetsParams() *ListAWSSubnetsParams {
	return &ListAWSSubnetsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListAWSSubnetsParamsWithTimeout creates a new ListAWSSubnetsParams object
// with the ability to set a timeout on a request.
func NewListAWSSubnetsParamsWithTimeout(timeout time.Duration) *ListAWSSubnetsParams {
	return &ListAWSSubnetsParams{
		timeout: timeout,
	}
}

// NewListAWSSubnetsParamsWithContext creates a new ListAWSSubnetsParams object
// with the ability to set a context for a request.
func NewListAWSSubnetsParamsWithContext(ctx context.Context) *ListAWSSubnetsParams {
	return &ListAWSSubnetsParams{
		Context: ctx,
	}
}

// NewListAWSSubnetsParamsWithHTTPClient creates a new ListAWSSubnetsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListAWSSubnetsParamsWithHTTPClient(client *http.Client) *ListAWSSubnetsParams {
	return &ListAWSSubnetsParams{
		HTTPClient: client,
	}
}

/* ListAWSSubnetsParams contains all the parameters to send to the API endpoint
   for the list a w s subnets operation.

   Typically these are written to a http.Request.
*/
type ListAWSSubnetsParams struct {

	// AccessKeyID.
	AccessKeyID *string

	// AssumeRoleARN.
	AssumeRoleARN *string

	// AssumeRoleExternalID.
	AssumeRoleExternalID *string

	// Credential.
	Credential *string

	// SecretAccessKey.
	SecretAccessKey *string

	// Dc.
	DC string

	// Vpc.
	VPC *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list a w s subnets params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAWSSubnetsParams) WithDefaults() *ListAWSSubnetsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list a w s subnets params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListAWSSubnetsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list a w s subnets params
func (o *ListAWSSubnetsParams) WithTimeout(timeout time.Duration) *ListAWSSubnetsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list a w s subnets params
func (o *ListAWSSubnetsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list a w s subnets params
func (o *ListAWSSubnetsParams) WithContext(ctx context.Context) *ListAWSSubnetsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list a w s subnets params
func (o *ListAWSSubnetsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list a w s subnets params
func (o *ListAWSSubnetsParams) WithHTTPClient(client *http.Client) *ListAWSSubnetsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list a w s subnets params
func (o *ListAWSSubnetsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAccessKeyID adds the accessKeyID to the list a w s subnets params
func (o *ListAWSSubnetsParams) WithAccessKeyID(accessKeyID *string) *ListAWSSubnetsParams {
	o.SetAccessKeyID(accessKeyID)
	return o
}

// SetAccessKeyID adds the accessKeyId to the list a w s subnets params
func (o *ListAWSSubnetsParams) SetAccessKeyID(accessKeyID *string) {
	o.AccessKeyID = accessKeyID
}

// WithAssumeRoleARN adds the assumeRoleARN to the list a w s subnets params
func (o *ListAWSSubnetsParams) WithAssumeRoleARN(assumeRoleARN *string) *ListAWSSubnetsParams {
	o.SetAssumeRoleARN(assumeRoleARN)
	return o
}

// SetAssumeRoleARN adds the assumeRoleARN to the list a w s subnets params
func (o *ListAWSSubnetsParams) SetAssumeRoleARN(assumeRoleARN *string) {
	o.AssumeRoleARN = assumeRoleARN
}

// WithAssumeRoleExternalID adds the assumeRoleExternalID to the list a w s subnets params
func (o *ListAWSSubnetsParams) WithAssumeRoleExternalID(assumeRoleExternalID *string) *ListAWSSubnetsParams {
	o.SetAssumeRoleExternalID(assumeRoleExternalID)
	return o
}

// SetAssumeRoleExternalID adds the assumeRoleExternalId to the list a w s subnets params
func (o *ListAWSSubnetsParams) SetAssumeRoleExternalID(assumeRoleExternalID *string) {
	o.AssumeRoleExternalID = assumeRoleExternalID
}

// WithCredential adds the credential to the list a w s subnets params
func (o *ListAWSSubnetsParams) WithCredential(credential *string) *ListAWSSubnetsParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list a w s subnets params
func (o *ListAWSSubnetsParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithSecretAccessKey adds the secretAccessKey to the list a w s subnets params
func (o *ListAWSSubnetsParams) WithSecretAccessKey(secretAccessKey *string) *ListAWSSubnetsParams {
	o.SetSecretAccessKey(secretAccessKey)
	return o
}

// SetSecretAccessKey adds the secretAccessKey to the list a w s subnets params
func (o *ListAWSSubnetsParams) SetSecretAccessKey(secretAccessKey *string) {
	o.SecretAccessKey = secretAccessKey
}

// WithDC adds the dc to the list a w s subnets params
func (o *ListAWSSubnetsParams) WithDC(dc string) *ListAWSSubnetsParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list a w s subnets params
func (o *ListAWSSubnetsParams) SetDC(dc string) {
	o.DC = dc
}

// WithVPC adds the vpc to the list a w s subnets params
func (o *ListAWSSubnetsParams) WithVPC(vpc *string) *ListAWSSubnetsParams {
	o.SetVPC(vpc)
	return o
}

// SetVPC adds the vpc to the list a w s subnets params
func (o *ListAWSSubnetsParams) SetVPC(vpc *string) {
	o.VPC = vpc
}

// WriteToRequest writes these params to a swagger request
func (o *ListAWSSubnetsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if o.AssumeRoleARN != nil {

		// header param AssumeRoleARN
		if err := r.SetHeaderParam("AssumeRoleARN", *o.AssumeRoleARN); err != nil {
			return err
		}
	}

	if o.AssumeRoleExternalID != nil {

		// header param AssumeRoleExternalID
		if err := r.SetHeaderParam("AssumeRoleExternalID", *o.AssumeRoleExternalID); err != nil {
			return err
		}
	}

	if o.Credential != nil {

		// header param Credential
		if err := r.SetHeaderParam("Credential", *o.Credential); err != nil {
			return err
		}
	}

	if o.SecretAccessKey != nil {

		// header param SecretAccessKey
		if err := r.SetHeaderParam("SecretAccessKey", *o.SecretAccessKey); err != nil {
			return err
		}
	}

	// path param dc
	if err := r.SetPathParam("dc", o.DC); err != nil {
		return err
	}

	if o.VPC != nil {

		// header param vpc
		if err := r.SetHeaderParam("vpc", *o.VPC); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
