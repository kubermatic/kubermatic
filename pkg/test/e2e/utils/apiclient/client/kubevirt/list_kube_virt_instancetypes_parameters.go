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

// NewListKubeVirtInstancetypesParams creates a new ListKubeVirtInstancetypesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListKubeVirtInstancetypesParams() *ListKubeVirtInstancetypesParams {
	return &ListKubeVirtInstancetypesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListKubeVirtInstancetypesParamsWithTimeout creates a new ListKubeVirtInstancetypesParams object
// with the ability to set a timeout on a request.
func NewListKubeVirtInstancetypesParamsWithTimeout(timeout time.Duration) *ListKubeVirtInstancetypesParams {
	return &ListKubeVirtInstancetypesParams{
		timeout: timeout,
	}
}

// NewListKubeVirtInstancetypesParamsWithContext creates a new ListKubeVirtInstancetypesParams object
// with the ability to set a context for a request.
func NewListKubeVirtInstancetypesParamsWithContext(ctx context.Context) *ListKubeVirtInstancetypesParams {
	return &ListKubeVirtInstancetypesParams{
		Context: ctx,
	}
}

// NewListKubeVirtInstancetypesParamsWithHTTPClient creates a new ListKubeVirtInstancetypesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListKubeVirtInstancetypesParamsWithHTTPClient(client *http.Client) *ListKubeVirtInstancetypesParams {
	return &ListKubeVirtInstancetypesParams{
		HTTPClient: client,
	}
}

/*
ListKubeVirtInstancetypesParams contains all the parameters to send to the API endpoint

	for the list kube virt instancetypes operation.

	Typically these are written to a http.Request.
*/
type ListKubeVirtInstancetypesParams struct {

	// Credential.
	Credential *string

	// Kubeconfig.
	Kubeconfig *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list kube virt instancetypes params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListKubeVirtInstancetypesParams) WithDefaults() *ListKubeVirtInstancetypesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list kube virt instancetypes params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListKubeVirtInstancetypesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list kube virt instancetypes params
func (o *ListKubeVirtInstancetypesParams) WithTimeout(timeout time.Duration) *ListKubeVirtInstancetypesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list kube virt instancetypes params
func (o *ListKubeVirtInstancetypesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list kube virt instancetypes params
func (o *ListKubeVirtInstancetypesParams) WithContext(ctx context.Context) *ListKubeVirtInstancetypesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list kube virt instancetypes params
func (o *ListKubeVirtInstancetypesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list kube virt instancetypes params
func (o *ListKubeVirtInstancetypesParams) WithHTTPClient(client *http.Client) *ListKubeVirtInstancetypesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list kube virt instancetypes params
func (o *ListKubeVirtInstancetypesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithCredential adds the credential to the list kube virt instancetypes params
func (o *ListKubeVirtInstancetypesParams) WithCredential(credential *string) *ListKubeVirtInstancetypesParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list kube virt instancetypes params
func (o *ListKubeVirtInstancetypesParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithKubeconfig adds the kubeconfig to the list kube virt instancetypes params
func (o *ListKubeVirtInstancetypesParams) WithKubeconfig(kubeconfig *string) *ListKubeVirtInstancetypesParams {
	o.SetKubeconfig(kubeconfig)
	return o
}

// SetKubeconfig adds the kubeconfig to the list kube virt instancetypes params
func (o *ListKubeVirtInstancetypesParams) SetKubeconfig(kubeconfig *string) {
	o.Kubeconfig = kubeconfig
}

// WriteToRequest writes these params to a swagger request
func (o *ListKubeVirtInstancetypesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
