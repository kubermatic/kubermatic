// Code generated by go-swagger; DO NOT EDIT.

package admin

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

// NewGetKubermaticCustomLinksParams creates a new GetKubermaticCustomLinksParams object
// with the default values initialized.
func NewGetKubermaticCustomLinksParams() *GetKubermaticCustomLinksParams {

	return &GetKubermaticCustomLinksParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetKubermaticCustomLinksParamsWithTimeout creates a new GetKubermaticCustomLinksParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetKubermaticCustomLinksParamsWithTimeout(timeout time.Duration) *GetKubermaticCustomLinksParams {

	return &GetKubermaticCustomLinksParams{

		timeout: timeout,
	}
}

// NewGetKubermaticCustomLinksParamsWithContext creates a new GetKubermaticCustomLinksParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetKubermaticCustomLinksParamsWithContext(ctx context.Context) *GetKubermaticCustomLinksParams {

	return &GetKubermaticCustomLinksParams{

		Context: ctx,
	}
}

// NewGetKubermaticCustomLinksParamsWithHTTPClient creates a new GetKubermaticCustomLinksParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetKubermaticCustomLinksParamsWithHTTPClient(client *http.Client) *GetKubermaticCustomLinksParams {

	return &GetKubermaticCustomLinksParams{
		HTTPClient: client,
	}
}

/*GetKubermaticCustomLinksParams contains all the parameters to send to the API endpoint
for the get kubermatic custom links operation typically these are written to a http.Request
*/
type GetKubermaticCustomLinksParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get kubermatic custom links params
func (o *GetKubermaticCustomLinksParams) WithTimeout(timeout time.Duration) *GetKubermaticCustomLinksParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get kubermatic custom links params
func (o *GetKubermaticCustomLinksParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get kubermatic custom links params
func (o *GetKubermaticCustomLinksParams) WithContext(ctx context.Context) *GetKubermaticCustomLinksParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get kubermatic custom links params
func (o *GetKubermaticCustomLinksParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get kubermatic custom links params
func (o *GetKubermaticCustomLinksParams) WithHTTPClient(client *http.Client) *GetKubermaticCustomLinksParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get kubermatic custom links params
func (o *GetKubermaticCustomLinksParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *GetKubermaticCustomLinksParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
