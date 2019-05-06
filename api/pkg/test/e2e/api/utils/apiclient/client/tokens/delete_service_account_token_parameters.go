// Code generated by go-swagger; DO NOT EDIT.

package tokens

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"

	strfmt "github.com/go-openapi/strfmt"
)

// NewDeleteServiceAccountTokenParams creates a new DeleteServiceAccountTokenParams object
// with the default values initialized.
func NewDeleteServiceAccountTokenParams() *DeleteServiceAccountTokenParams {
	var ()
	return &DeleteServiceAccountTokenParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteServiceAccountTokenParamsWithTimeout creates a new DeleteServiceAccountTokenParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewDeleteServiceAccountTokenParamsWithTimeout(timeout time.Duration) *DeleteServiceAccountTokenParams {
	var ()
	return &DeleteServiceAccountTokenParams{

		timeout: timeout,
	}
}

// NewDeleteServiceAccountTokenParamsWithContext creates a new DeleteServiceAccountTokenParams object
// with the default values initialized, and the ability to set a context for a request
func NewDeleteServiceAccountTokenParamsWithContext(ctx context.Context) *DeleteServiceAccountTokenParams {
	var ()
	return &DeleteServiceAccountTokenParams{

		Context: ctx,
	}
}

// NewDeleteServiceAccountTokenParamsWithHTTPClient creates a new DeleteServiceAccountTokenParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewDeleteServiceAccountTokenParamsWithHTTPClient(client *http.Client) *DeleteServiceAccountTokenParams {
	var ()
	return &DeleteServiceAccountTokenParams{
		HTTPClient: client,
	}
}

/*DeleteServiceAccountTokenParams contains all the parameters to send to the API endpoint
for the delete service account token operation typically these are written to a http.Request
*/
type DeleteServiceAccountTokenParams struct {

	/*ProjectID*/
	ProjectID string
	/*ServiceaccountID*/
	ServiceaccountID string
	/*TokenID*/
	TokenID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the delete service account token params
func (o *DeleteServiceAccountTokenParams) WithTimeout(timeout time.Duration) *DeleteServiceAccountTokenParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete service account token params
func (o *DeleteServiceAccountTokenParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete service account token params
func (o *DeleteServiceAccountTokenParams) WithContext(ctx context.Context) *DeleteServiceAccountTokenParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete service account token params
func (o *DeleteServiceAccountTokenParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete service account token params
func (o *DeleteServiceAccountTokenParams) WithHTTPClient(client *http.Client) *DeleteServiceAccountTokenParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete service account token params
func (o *DeleteServiceAccountTokenParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithProjectID adds the projectID to the delete service account token params
func (o *DeleteServiceAccountTokenParams) WithProjectID(projectID string) *DeleteServiceAccountTokenParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the delete service account token params
func (o *DeleteServiceAccountTokenParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithServiceaccountID adds the serviceaccountID to the delete service account token params
func (o *DeleteServiceAccountTokenParams) WithServiceaccountID(serviceaccountID string) *DeleteServiceAccountTokenParams {
	o.SetServiceaccountID(serviceaccountID)
	return o
}

// SetServiceaccountID adds the serviceaccountId to the delete service account token params
func (o *DeleteServiceAccountTokenParams) SetServiceaccountID(serviceaccountID string) {
	o.ServiceaccountID = serviceaccountID
}

// WithTokenID adds the tokenID to the delete service account token params
func (o *DeleteServiceAccountTokenParams) WithTokenID(tokenID string) *DeleteServiceAccountTokenParams {
	o.SetTokenID(tokenID)
	return o
}

// SetTokenID adds the tokenId to the delete service account token params
func (o *DeleteServiceAccountTokenParams) SetTokenID(tokenID string) {
	o.TokenID = tokenID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteServiceAccountTokenParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	// path param serviceaccount_id
	if err := r.SetPathParam("serviceaccount_id", o.ServiceaccountID); err != nil {
		return err
	}

	// path param token_id
	if err := r.SetPathParam("token_id", o.TokenID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
