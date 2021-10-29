// Code generated by go-swagger; DO NOT EDIT.

package project

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

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// NewCreateExternalClusterParams creates a new CreateExternalClusterParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewCreateExternalClusterParams() *CreateExternalClusterParams {
	return &CreateExternalClusterParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewCreateExternalClusterParamsWithTimeout creates a new CreateExternalClusterParams object
// with the ability to set a timeout on a request.
func NewCreateExternalClusterParamsWithTimeout(timeout time.Duration) *CreateExternalClusterParams {
	return &CreateExternalClusterParams{
		timeout: timeout,
	}
}

// NewCreateExternalClusterParamsWithContext creates a new CreateExternalClusterParams object
// with the ability to set a context for a request.
func NewCreateExternalClusterParamsWithContext(ctx context.Context) *CreateExternalClusterParams {
	return &CreateExternalClusterParams{
		Context: ctx,
	}
}

// NewCreateExternalClusterParamsWithHTTPClient creates a new CreateExternalClusterParams object
// with the ability to set a custom HTTPClient for a request.
func NewCreateExternalClusterParamsWithHTTPClient(client *http.Client) *CreateExternalClusterParams {
	return &CreateExternalClusterParams{
		HTTPClient: client,
	}
}

/* CreateExternalClusterParams contains all the parameters to send to the API endpoint
   for the create external cluster operation.

   Typically these are written to a http.Request.
*/
type CreateExternalClusterParams struct {

	// Body.
	Body *models.Body

	/* Credential.

	   The credential name used in the preset for the provider
	*/
	Credential *string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the create external cluster params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateExternalClusterParams) WithDefaults() *CreateExternalClusterParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the create external cluster params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateExternalClusterParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the create external cluster params
func (o *CreateExternalClusterParams) WithTimeout(timeout time.Duration) *CreateExternalClusterParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the create external cluster params
func (o *CreateExternalClusterParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the create external cluster params
func (o *CreateExternalClusterParams) WithContext(ctx context.Context) *CreateExternalClusterParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the create external cluster params
func (o *CreateExternalClusterParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the create external cluster params
func (o *CreateExternalClusterParams) WithHTTPClient(client *http.Client) *CreateExternalClusterParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the create external cluster params
func (o *CreateExternalClusterParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the create external cluster params
func (o *CreateExternalClusterParams) WithBody(body *models.Body) *CreateExternalClusterParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the create external cluster params
func (o *CreateExternalClusterParams) SetBody(body *models.Body) {
	o.Body = body
}

// WithCredential adds the credential to the create external cluster params
func (o *CreateExternalClusterParams) WithCredential(credential *string) *CreateExternalClusterParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the create external cluster params
func (o *CreateExternalClusterParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithProjectID adds the projectID to the create external cluster params
func (o *CreateExternalClusterParams) WithProjectID(projectID string) *CreateExternalClusterParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the create external cluster params
func (o *CreateExternalClusterParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *CreateExternalClusterParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	if o.Credential != nil {

		// header param Credential
		if err := r.SetHeaderParam("Credential", *o.Credential); err != nil {
			return err
		}
	}

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
