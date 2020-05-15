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

// NewListAWSSizesParams creates a new ListAWSSizesParams object
// with the default values initialized.
func NewListAWSSizesParams() *ListAWSSizesParams {
	var ()
	return &ListAWSSizesParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewListAWSSizesParamsWithTimeout creates a new ListAWSSizesParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListAWSSizesParamsWithTimeout(timeout time.Duration) *ListAWSSizesParams {
	var ()
	return &ListAWSSizesParams{

		timeout: timeout,
	}
}

// NewListAWSSizesParamsWithContext creates a new ListAWSSizesParams object
// with the default values initialized, and the ability to set a context for a request
func NewListAWSSizesParamsWithContext(ctx context.Context) *ListAWSSizesParams {
	var ()
	return &ListAWSSizesParams{

		Context: ctx,
	}
}

// NewListAWSSizesParamsWithHTTPClient creates a new ListAWSSizesParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListAWSSizesParamsWithHTTPClient(client *http.Client) *ListAWSSizesParams {
	var ()
	return &ListAWSSizesParams{
		HTTPClient: client,
	}
}

/*ListAWSSizesParams contains all the parameters to send to the API endpoint
for the list a w s sizes operation typically these are written to a http.Request
*/
type ListAWSSizesParams struct {

	/*Region*/
	Region *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list a w s sizes params
func (o *ListAWSSizesParams) WithTimeout(timeout time.Duration) *ListAWSSizesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list a w s sizes params
func (o *ListAWSSizesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list a w s sizes params
func (o *ListAWSSizesParams) WithContext(ctx context.Context) *ListAWSSizesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list a w s sizes params
func (o *ListAWSSizesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list a w s sizes params
func (o *ListAWSSizesParams) WithHTTPClient(client *http.Client) *ListAWSSizesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list a w s sizes params
func (o *ListAWSSizesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithRegion adds the region to the list a w s sizes params
func (o *ListAWSSizesParams) WithRegion(region *string) *ListAWSSizesParams {
	o.SetRegion(region)
	return o
}

// SetRegion adds the region to the list a w s sizes params
func (o *ListAWSSizesParams) SetRegion(region *string) {
	o.Region = region
}

// WriteToRequest writes these params to a swagger request
func (o *ListAWSSizesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Region != nil {

		// header param Region
		if err := r.SetHeaderParam("Region", *o.Region); err != nil {
			return err
		}

	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
