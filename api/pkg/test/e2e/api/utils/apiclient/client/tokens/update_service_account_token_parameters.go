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
	"github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// NewUpdateServiceAccountTokenParams creates a new UpdateServiceAccountTokenParams object
// with the default values initialized.
func NewUpdateServiceAccountTokenParams() *UpdateServiceAccountTokenParams {
	var ()
	return &UpdateServiceAccountTokenParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewUpdateServiceAccountTokenParamsWithTimeout creates a new UpdateServiceAccountTokenParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewUpdateServiceAccountTokenParamsWithTimeout(timeout time.Duration) *UpdateServiceAccountTokenParams {
	var ()
	return &UpdateServiceAccountTokenParams{

		timeout: timeout,
	}
}

// NewUpdateServiceAccountTokenParamsWithContext creates a new UpdateServiceAccountTokenParams object
// with the default values initialized, and the ability to set a context for a request
func NewUpdateServiceAccountTokenParamsWithContext(ctx context.Context) *UpdateServiceAccountTokenParams {
	var ()
	return &UpdateServiceAccountTokenParams{

		Context: ctx,
	}
}

// NewUpdateServiceAccountTokenParamsWithHTTPClient creates a new UpdateServiceAccountTokenParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewUpdateServiceAccountTokenParamsWithHTTPClient(client *http.Client) *UpdateServiceAccountTokenParams {
	var ()
	return &UpdateServiceAccountTokenParams{
		HTTPClient: client,
	}
}

/*UpdateServiceAccountTokenParams contains all the parameters to send to the API endpoint
for the update service account token operation typically these are written to a http.Request
*/
type UpdateServiceAccountTokenParams struct {

	/*Body*/
	Body *models.PublicServiceAccountToken
	/*ProjectID*/
	ProjectID string
	/*ServiceaccountID*/
	ServiceAccountID string
	/*TokenID*/
	TokenID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the update service account token params
func (o *UpdateServiceAccountTokenParams) WithTimeout(timeout time.Duration) *UpdateServiceAccountTokenParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the update service account token params
func (o *UpdateServiceAccountTokenParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the update service account token params
func (o *UpdateServiceAccountTokenParams) WithContext(ctx context.Context) *UpdateServiceAccountTokenParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the update service account token params
func (o *UpdateServiceAccountTokenParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the update service account token params
func (o *UpdateServiceAccountTokenParams) WithHTTPClient(client *http.Client) *UpdateServiceAccountTokenParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the update service account token params
func (o *UpdateServiceAccountTokenParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the update service account token params
func (o *UpdateServiceAccountTokenParams) WithBody(body *models.PublicServiceAccountToken) *UpdateServiceAccountTokenParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the update service account token params
func (o *UpdateServiceAccountTokenParams) SetBody(body *models.PublicServiceAccountToken) {
	o.Body = body
}

// WithProjectID adds the projectID to the update service account token params
func (o *UpdateServiceAccountTokenParams) WithProjectID(projectID string) *UpdateServiceAccountTokenParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the update service account token params
func (o *UpdateServiceAccountTokenParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithServiceAccountID adds the serviceaccountID to the update service account token params
func (o *UpdateServiceAccountTokenParams) WithServiceAccountID(serviceaccountID string) *UpdateServiceAccountTokenParams {
	o.SetServiceAccountID(serviceaccountID)
	return o
}

// SetServiceAccountID adds the serviceaccountId to the update service account token params
func (o *UpdateServiceAccountTokenParams) SetServiceAccountID(serviceaccountID string) {
	o.ServiceAccountID = serviceaccountID
}

// WithTokenID adds the tokenID to the update service account token params
func (o *UpdateServiceAccountTokenParams) WithTokenID(tokenID string) *UpdateServiceAccountTokenParams {
	o.SetTokenID(tokenID)
	return o
}

// SetTokenID adds the tokenId to the update service account token params
func (o *UpdateServiceAccountTokenParams) SetTokenID(tokenID string) {
	o.TokenID = tokenID
}

// WriteToRequest writes these params to a swagger request
func (o *UpdateServiceAccountTokenParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	// path param serviceaccount_id
	if err := r.SetPathParam("serviceaccount_id", o.ServiceAccountID); err != nil {
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
