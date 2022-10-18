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
)

// NewListServiceAccountTokensParams creates a new ListServiceAccountTokensParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListServiceAccountTokensParams() *ListServiceAccountTokensParams {
	return &ListServiceAccountTokensParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListServiceAccountTokensParamsWithTimeout creates a new ListServiceAccountTokensParams object
// with the ability to set a timeout on a request.
func NewListServiceAccountTokensParamsWithTimeout(timeout time.Duration) *ListServiceAccountTokensParams {
	return &ListServiceAccountTokensParams{
		timeout: timeout,
	}
}

// NewListServiceAccountTokensParamsWithContext creates a new ListServiceAccountTokensParams object
// with the ability to set a context for a request.
func NewListServiceAccountTokensParamsWithContext(ctx context.Context) *ListServiceAccountTokensParams {
	return &ListServiceAccountTokensParams{
		Context: ctx,
	}
}

// NewListServiceAccountTokensParamsWithHTTPClient creates a new ListServiceAccountTokensParams object
// with the ability to set a custom HTTPClient for a request.
func NewListServiceAccountTokensParamsWithHTTPClient(client *http.Client) *ListServiceAccountTokensParams {
	return &ListServiceAccountTokensParams{
		HTTPClient: client,
	}
}

/*
ListServiceAccountTokensParams contains all the parameters to send to the API endpoint

	for the list service account tokens operation.

	Typically these are written to a http.Request.
*/
type ListServiceAccountTokensParams struct {

	// ProjectID.
	ProjectID string

	// ServiceaccountID.
	ServiceAccountID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list service account tokens params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListServiceAccountTokensParams) WithDefaults() *ListServiceAccountTokensParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list service account tokens params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListServiceAccountTokensParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list service account tokens params
func (o *ListServiceAccountTokensParams) WithTimeout(timeout time.Duration) *ListServiceAccountTokensParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list service account tokens params
func (o *ListServiceAccountTokensParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list service account tokens params
func (o *ListServiceAccountTokensParams) WithContext(ctx context.Context) *ListServiceAccountTokensParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list service account tokens params
func (o *ListServiceAccountTokensParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list service account tokens params
func (o *ListServiceAccountTokensParams) WithHTTPClient(client *http.Client) *ListServiceAccountTokensParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list service account tokens params
func (o *ListServiceAccountTokensParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithProjectID adds the projectID to the list service account tokens params
func (o *ListServiceAccountTokensParams) WithProjectID(projectID string) *ListServiceAccountTokensParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list service account tokens params
func (o *ListServiceAccountTokensParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithServiceAccountID adds the serviceaccountID to the list service account tokens params
func (o *ListServiceAccountTokensParams) WithServiceAccountID(serviceaccountID string) *ListServiceAccountTokensParams {
	o.SetServiceAccountID(serviceaccountID)
	return o
}

// SetServiceAccountID adds the serviceaccountId to the list service account tokens params
func (o *ListServiceAccountTokensParams) SetServiceAccountID(serviceaccountID string) {
	o.ServiceAccountID = serviceaccountID
}

// WriteToRequest writes these params to a swagger request
func (o *ListServiceAccountTokensParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	// path param serviceaccount_id
	if err := r.SetPathParam("serviceaccount_id", o.ServiceAccountID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
