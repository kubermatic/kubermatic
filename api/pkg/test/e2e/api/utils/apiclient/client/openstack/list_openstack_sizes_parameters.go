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
)

// NewListOpenstackSizesParams creates a new ListOpenstackSizesParams object
// with the default values initialized.
func NewListOpenstackSizesParams() *ListOpenstackSizesParams {

	return &ListOpenstackSizesParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewListOpenstackSizesParamsWithTimeout creates a new ListOpenstackSizesParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListOpenstackSizesParamsWithTimeout(timeout time.Duration) *ListOpenstackSizesParams {

	return &ListOpenstackSizesParams{

		timeout: timeout,
	}
}

// NewListOpenstackSizesParamsWithContext creates a new ListOpenstackSizesParams object
// with the default values initialized, and the ability to set a context for a request
func NewListOpenstackSizesParamsWithContext(ctx context.Context) *ListOpenstackSizesParams {

	return &ListOpenstackSizesParams{

		Context: ctx,
	}
}

// NewListOpenstackSizesParamsWithHTTPClient creates a new ListOpenstackSizesParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListOpenstackSizesParamsWithHTTPClient(client *http.Client) *ListOpenstackSizesParams {

	return &ListOpenstackSizesParams{
		HTTPClient: client,
	}
}

/*ListOpenstackSizesParams contains all the parameters to send to the API endpoint
for the list openstack sizes operation typically these are written to a http.Request
*/
type ListOpenstackSizesParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list openstack sizes params
func (o *ListOpenstackSizesParams) WithTimeout(timeout time.Duration) *ListOpenstackSizesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list openstack sizes params
func (o *ListOpenstackSizesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list openstack sizes params
func (o *ListOpenstackSizesParams) WithContext(ctx context.Context) *ListOpenstackSizesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list openstack sizes params
func (o *ListOpenstackSizesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list openstack sizes params
func (o *ListOpenstackSizesParams) WithHTTPClient(client *http.Client) *ListOpenstackSizesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list openstack sizes params
func (o *ListOpenstackSizesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListOpenstackSizesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
