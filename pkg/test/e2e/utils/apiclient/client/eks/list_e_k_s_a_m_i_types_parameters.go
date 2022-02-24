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

// NewListEKSAMITypesParams creates a new ListEKSAMITypesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListEKSAMITypesParams() *ListEKSAMITypesParams {
	return &ListEKSAMITypesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListEKSAMITypesParamsWithTimeout creates a new ListEKSAMITypesParams object
// with the ability to set a timeout on a request.
func NewListEKSAMITypesParamsWithTimeout(timeout time.Duration) *ListEKSAMITypesParams {
	return &ListEKSAMITypesParams{
		timeout: timeout,
	}
}

// NewListEKSAMITypesParamsWithContext creates a new ListEKSAMITypesParams object
// with the ability to set a context for a request.
func NewListEKSAMITypesParamsWithContext(ctx context.Context) *ListEKSAMITypesParams {
	return &ListEKSAMITypesParams{
		Context: ctx,
	}
}

// NewListEKSAMITypesParamsWithHTTPClient creates a new ListEKSAMITypesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListEKSAMITypesParamsWithHTTPClient(client *http.Client) *ListEKSAMITypesParams {
	return &ListEKSAMITypesParams{
		HTTPClient: client,
	}
}

/* ListEKSAMITypesParams contains all the parameters to send to the API endpoint
   for the list e k s a m i types operation.

   Typically these are written to a http.Request.
*/
type ListEKSAMITypesParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list e k s a m i types params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListEKSAMITypesParams) WithDefaults() *ListEKSAMITypesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list e k s a m i types params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListEKSAMITypesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list e k s a m i types params
func (o *ListEKSAMITypesParams) WithTimeout(timeout time.Duration) *ListEKSAMITypesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list e k s a m i types params
func (o *ListEKSAMITypesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list e k s a m i types params
func (o *ListEKSAMITypesParams) WithContext(ctx context.Context) *ListEKSAMITypesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list e k s a m i types params
func (o *ListEKSAMITypesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list e k s a m i types params
func (o *ListEKSAMITypesParams) WithHTTPClient(client *http.Client) *ListEKSAMITypesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list e k s a m i types params
func (o *ListEKSAMITypesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListEKSAMITypesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
