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

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// NewUpdateAdmissionPluginParams creates a new UpdateAdmissionPluginParams object
// with the default values initialized.
func NewUpdateAdmissionPluginParams() *UpdateAdmissionPluginParams {
	var ()
	return &UpdateAdmissionPluginParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewUpdateAdmissionPluginParamsWithTimeout creates a new UpdateAdmissionPluginParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewUpdateAdmissionPluginParamsWithTimeout(timeout time.Duration) *UpdateAdmissionPluginParams {
	var ()
	return &UpdateAdmissionPluginParams{

		timeout: timeout,
	}
}

// NewUpdateAdmissionPluginParamsWithContext creates a new UpdateAdmissionPluginParams object
// with the default values initialized, and the ability to set a context for a request
func NewUpdateAdmissionPluginParamsWithContext(ctx context.Context) *UpdateAdmissionPluginParams {
	var ()
	return &UpdateAdmissionPluginParams{

		Context: ctx,
	}
}

// NewUpdateAdmissionPluginParamsWithHTTPClient creates a new UpdateAdmissionPluginParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewUpdateAdmissionPluginParamsWithHTTPClient(client *http.Client) *UpdateAdmissionPluginParams {
	var ()
	return &UpdateAdmissionPluginParams{
		HTTPClient: client,
	}
}

/*UpdateAdmissionPluginParams contains all the parameters to send to the API endpoint
for the update admission plugin operation typically these are written to a http.Request
*/
type UpdateAdmissionPluginParams struct {

	/*Body*/
	Body *models.AdmissionPlugin
	/*Name*/
	Name string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the update admission plugin params
func (o *UpdateAdmissionPluginParams) WithTimeout(timeout time.Duration) *UpdateAdmissionPluginParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the update admission plugin params
func (o *UpdateAdmissionPluginParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the update admission plugin params
func (o *UpdateAdmissionPluginParams) WithContext(ctx context.Context) *UpdateAdmissionPluginParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the update admission plugin params
func (o *UpdateAdmissionPluginParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the update admission plugin params
func (o *UpdateAdmissionPluginParams) WithHTTPClient(client *http.Client) *UpdateAdmissionPluginParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the update admission plugin params
func (o *UpdateAdmissionPluginParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the update admission plugin params
func (o *UpdateAdmissionPluginParams) WithBody(body *models.AdmissionPlugin) *UpdateAdmissionPluginParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the update admission plugin params
func (o *UpdateAdmissionPluginParams) SetBody(body *models.AdmissionPlugin) {
	o.Body = body
}

// WithName adds the name to the update admission plugin params
func (o *UpdateAdmissionPluginParams) WithName(name string) *UpdateAdmissionPluginParams {
	o.SetName(name)
	return o
}

// SetName adds the name to the update admission plugin params
func (o *UpdateAdmissionPluginParams) SetName(name string) {
	o.Name = name
}

// WriteToRequest writes these params to a swagger request
func (o *UpdateAdmissionPluginParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	// path param name
	if err := r.SetPathParam("name", o.Name); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
