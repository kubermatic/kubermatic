// Code generated by go-swagger; DO NOT EDIT.

package azure

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

// NewListAzureAvailabilityZonesParams creates a new ListAzureAvailabilityZonesParams object
// with the default values initialized.
func NewListAzureAvailabilityZonesParams() *ListAzureAvailabilityZonesParams {

	return &ListAzureAvailabilityZonesParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewListAzureAvailabilityZonesParamsWithTimeout creates a new ListAzureAvailabilityZonesParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListAzureAvailabilityZonesParamsWithTimeout(timeout time.Duration) *ListAzureAvailabilityZonesParams {

	return &ListAzureAvailabilityZonesParams{

		timeout: timeout,
	}
}

// NewListAzureAvailabilityZonesParamsWithContext creates a new ListAzureAvailabilityZonesParams object
// with the default values initialized, and the ability to set a context for a request
func NewListAzureAvailabilityZonesParamsWithContext(ctx context.Context) *ListAzureAvailabilityZonesParams {

	return &ListAzureAvailabilityZonesParams{

		Context: ctx,
	}
}

// NewListAzureAvailabilityZonesParamsWithHTTPClient creates a new ListAzureAvailabilityZonesParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListAzureAvailabilityZonesParamsWithHTTPClient(client *http.Client) *ListAzureAvailabilityZonesParams {

	return &ListAzureAvailabilityZonesParams{
		HTTPClient: client,
	}
}

/*ListAzureAvailabilityZonesParams contains all the parameters to send to the API endpoint
for the list azure availability zones operation typically these are written to a http.Request
*/
type ListAzureAvailabilityZonesParams struct {
	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list azure availability zones params
func (o *ListAzureAvailabilityZonesParams) WithTimeout(timeout time.Duration) *ListAzureAvailabilityZonesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list azure availability zones params
func (o *ListAzureAvailabilityZonesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list azure availability zones params
func (o *ListAzureAvailabilityZonesParams) WithContext(ctx context.Context) *ListAzureAvailabilityZonesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list azure availability zones params
func (o *ListAzureAvailabilityZonesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list azure availability zones params
func (o *ListAzureAvailabilityZonesParams) WithHTTPClient(client *http.Client) *ListAzureAvailabilityZonesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list azure availability zones params
func (o *ListAzureAvailabilityZonesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WriteToRequest writes these params to a swagger request
func (o *ListAzureAvailabilityZonesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
