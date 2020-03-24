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

// NewListAWSVPCSParams creates a new ListAWSVPCSParams object
// with the default values initialized.
func NewListAWSVPCSParams() *ListAWSVPCSParams {
	var ()
	return &ListAWSVPCSParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewListAWSVPCSParamsWithTimeout creates a new ListAWSVPCSParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListAWSVPCSParamsWithTimeout(timeout time.Duration) *ListAWSVPCSParams {
	var ()
	return &ListAWSVPCSParams{

		timeout: timeout,
	}
}

// NewListAWSVPCSParamsWithContext creates a new ListAWSVPCSParams object
// with the default values initialized, and the ability to set a context for a request
func NewListAWSVPCSParamsWithContext(ctx context.Context) *ListAWSVPCSParams {
	var ()
	return &ListAWSVPCSParams{

		Context: ctx,
	}
}

// NewListAWSVPCSParamsWithHTTPClient creates a new ListAWSVPCSParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListAWSVPCSParamsWithHTTPClient(client *http.Client) *ListAWSVPCSParams {
	var ()
	return &ListAWSVPCSParams{
		HTTPClient: client,
	}
}

/*ListAWSVPCSParams contains all the parameters to send to the API endpoint
for the list a w s v p c s operation typically these are written to a http.Request
*/
type ListAWSVPCSParams struct {

	/*AccessKeyID*/
	AccessKeyID *string
	/*Credential*/
	Credential *string
	/*SecretAccessKey*/
	SecretAccessKey *string
	/*Dc*/
	DC string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list a w s v p c s params
func (o *ListAWSVPCSParams) WithTimeout(timeout time.Duration) *ListAWSVPCSParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list a w s v p c s params
func (o *ListAWSVPCSParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list a w s v p c s params
func (o *ListAWSVPCSParams) WithContext(ctx context.Context) *ListAWSVPCSParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list a w s v p c s params
func (o *ListAWSVPCSParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list a w s v p c s params
func (o *ListAWSVPCSParams) WithHTTPClient(client *http.Client) *ListAWSVPCSParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list a w s v p c s params
func (o *ListAWSVPCSParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAccessKeyID adds the accessKeyID to the list a w s v p c s params
func (o *ListAWSVPCSParams) WithAccessKeyID(accessKeyID *string) *ListAWSVPCSParams {
	o.SetAccessKeyID(accessKeyID)
	return o
}

// SetAccessKeyID adds the accessKeyId to the list a w s v p c s params
func (o *ListAWSVPCSParams) SetAccessKeyID(accessKeyID *string) {
	o.AccessKeyID = accessKeyID
}

// WithCredential adds the credential to the list a w s v p c s params
func (o *ListAWSVPCSParams) WithCredential(credential *string) *ListAWSVPCSParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list a w s v p c s params
func (o *ListAWSVPCSParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithSecretAccessKey adds the secretAccessKey to the list a w s v p c s params
func (o *ListAWSVPCSParams) WithSecretAccessKey(secretAccessKey *string) *ListAWSVPCSParams {
	o.SetSecretAccessKey(secretAccessKey)
	return o
}

// SetSecretAccessKey adds the secretAccessKey to the list a w s v p c s params
func (o *ListAWSVPCSParams) SetSecretAccessKey(secretAccessKey *string) {
	o.SecretAccessKey = secretAccessKey
}

// WithDC adds the dc to the list a w s v p c s params
func (o *ListAWSVPCSParams) WithDC(dc string) *ListAWSVPCSParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list a w s v p c s params
func (o *ListAWSVPCSParams) SetDC(dc string) {
	o.DC = dc
}

// WriteToRequest writes these params to a swagger request
func (o *ListAWSVPCSParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
