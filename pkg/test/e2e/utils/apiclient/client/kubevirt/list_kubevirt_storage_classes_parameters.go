// Code generated by go-swagger; DO NOT EDIT.

package kubevirt

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

// NewListKubevirtStorageClassesParams creates a new ListKubevirtStorageClassesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListKubevirtStorageClassesParams() *ListKubevirtStorageClassesParams {
	return &ListKubevirtStorageClassesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListKubevirtStorageClassesParamsWithTimeout creates a new ListKubevirtStorageClassesParams object
// with the ability to set a timeout on a request.
func NewListKubevirtStorageClassesParamsWithTimeout(timeout time.Duration) *ListKubevirtStorageClassesParams {
	return &ListKubevirtStorageClassesParams{
		timeout: timeout,
	}
}

// NewListKubevirtStorageClassesParamsWithContext creates a new ListKubevirtStorageClassesParams object
// with the ability to set a context for a request.
func NewListKubevirtStorageClassesParamsWithContext(ctx context.Context) *ListKubevirtStorageClassesParams {
	return &ListKubevirtStorageClassesParams{
		Context: ctx,
	}
}

// NewListKubevirtStorageClassesParamsWithHTTPClient creates a new ListKubevirtStorageClassesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListKubevirtStorageClassesParamsWithHTTPClient(client *http.Client) *ListKubevirtStorageClassesParams {
	return &ListKubevirtStorageClassesParams{
		HTTPClient: client,
	}
}

/* ListKubevirtStorageClassesParams contains all the parameters to send to the API endpoint
   for the list kubevirt storage classes operation.

   Typically these are written to a http.Request.
*/
type ListKubevirtStorageClassesParams struct {

	// Credential.
	Credential *string

	// Kubeconfig.
	Kubeconfig *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list kubevirt storage classes params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListKubevirtStorageClassesParams) WithDefaults() *ListKubevirtStorageClassesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list kubevirt storage classes params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListKubevirtStorageClassesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list kubevirt storage classes params
func (o *ListKubevirtStorageClassesParams) WithTimeout(timeout time.Duration) *ListKubevirtStorageClassesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list kubevirt storage classes params
func (o *ListKubevirtStorageClassesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list kubevirt storage classes params
func (o *ListKubevirtStorageClassesParams) WithContext(ctx context.Context) *ListKubevirtStorageClassesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list kubevirt storage classes params
func (o *ListKubevirtStorageClassesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list kubevirt storage classes params
func (o *ListKubevirtStorageClassesParams) WithHTTPClient(client *http.Client) *ListKubevirtStorageClassesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list kubevirt storage classes params
func (o *ListKubevirtStorageClassesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCredential adds the credential to the list kubevirt storage classes params
func (o *ListKubevirtStorageClassesParams) WithCredential(credential *string) *ListKubevirtStorageClassesParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list kubevirt storage classes params
func (o *ListKubevirtStorageClassesParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithKubeconfig adds the kubeconfig to the list kubevirt storage classes params
func (o *ListKubevirtStorageClassesParams) WithKubeconfig(kubeconfig *string) *ListKubevirtStorageClassesParams {
	o.SetKubeconfig(kubeconfig)
	return o
}

// SetKubeconfig adds the kubeconfig to the list kubevirt storage classes params
func (o *ListKubevirtStorageClassesParams) SetKubeconfig(kubeconfig *string) {
	o.Kubeconfig = kubeconfig
}

// WriteToRequest writes these params to a swagger request
func (o *ListKubevirtStorageClassesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if o.Kubeconfig != nil {

		// header param Kubeconfig
		if err := r.SetHeaderParam("Kubeconfig", *o.Kubeconfig); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
