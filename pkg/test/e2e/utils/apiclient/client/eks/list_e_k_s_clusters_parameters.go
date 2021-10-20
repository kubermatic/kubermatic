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

// NewListEKSClustersParams creates a new ListEKSClustersParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListEKSClustersParams() *ListEKSClustersParams {
	return &ListEKSClustersParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListEKSClustersParamsWithTimeout creates a new ListEKSClustersParams object
// with the ability to set a timeout on a request.
func NewListEKSClustersParamsWithTimeout(timeout time.Duration) *ListEKSClustersParams {
	return &ListEKSClustersParams{
		timeout: timeout,
	}
}

// NewListEKSClustersParamsWithContext creates a new ListEKSClustersParams object
// with the ability to set a context for a request.
func NewListEKSClustersParamsWithContext(ctx context.Context) *ListEKSClustersParams {
	return &ListEKSClustersParams{
		Context: ctx,
	}
}

// NewListEKSClustersParamsWithHTTPClient creates a new ListEKSClustersParams object
// with the ability to set a custom HTTPClient for a request.
func NewListEKSClustersParamsWithHTTPClient(client *http.Client) *ListEKSClustersParams {
	return &ListEKSClustersParams{
		HTTPClient: client,
	}
}

/* ListEKSClustersParams contains all the parameters to send to the API endpoint
   for the list e k s clusters operation.

   Typically these are written to a http.Request.
*/
type ListEKSClustersParams struct {

	// AccessKeyID.
	AccessKeyID *string

	// Credential.
	Credential *string

	// Region.
	Region *string

	// SecretAccessKey.
	SecretAccessKey *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list e k s clusters params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListEKSClustersParams) WithDefaults() *ListEKSClustersParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list e k s clusters params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListEKSClustersParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list e k s clusters params
func (o *ListEKSClustersParams) WithTimeout(timeout time.Duration) *ListEKSClustersParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list e k s clusters params
func (o *ListEKSClustersParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list e k s clusters params
func (o *ListEKSClustersParams) WithContext(ctx context.Context) *ListEKSClustersParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list e k s clusters params
func (o *ListEKSClustersParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list e k s clusters params
func (o *ListEKSClustersParams) WithHTTPClient(client *http.Client) *ListEKSClustersParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list e k s clusters params
func (o *ListEKSClustersParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAccessKeyID adds the accessKeyID to the list e k s clusters params
func (o *ListEKSClustersParams) WithAccessKeyID(accessKeyID *string) *ListEKSClustersParams {
	o.SetAccessKeyID(accessKeyID)
	return o
}

// SetAccessKeyID adds the accessKeyId to the list e k s clusters params
func (o *ListEKSClustersParams) SetAccessKeyID(accessKeyID *string) {
	o.AccessKeyID = accessKeyID
}

// WithCredential adds the credential to the list e k s clusters params
func (o *ListEKSClustersParams) WithCredential(credential *string) *ListEKSClustersParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list e k s clusters params
func (o *ListEKSClustersParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithRegion adds the region to the list e k s clusters params
func (o *ListEKSClustersParams) WithRegion(region *string) *ListEKSClustersParams {
	o.SetRegion(region)
	return o
}

// SetRegion adds the region to the list e k s clusters params
func (o *ListEKSClustersParams) SetRegion(region *string) {
	o.Region = region
}

// WithSecretAccessKey adds the secretAccessKey to the list e k s clusters params
func (o *ListEKSClustersParams) WithSecretAccessKey(secretAccessKey *string) *ListEKSClustersParams {
	o.SetSecretAccessKey(secretAccessKey)
	return o
}

// SetSecretAccessKey adds the secretAccessKey to the list e k s clusters params
func (o *ListEKSClustersParams) SetSecretAccessKey(secretAccessKey *string) {
	o.SecretAccessKey = secretAccessKey
}

// WriteToRequest writes these params to a swagger request
func (o *ListEKSClustersParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
