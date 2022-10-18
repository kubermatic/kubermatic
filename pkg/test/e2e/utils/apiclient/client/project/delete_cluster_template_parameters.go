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
)

// NewDeleteClusterTemplateParams creates a new DeleteClusterTemplateParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteClusterTemplateParams() *DeleteClusterTemplateParams {
	return &DeleteClusterTemplateParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteClusterTemplateParamsWithTimeout creates a new DeleteClusterTemplateParams object
// with the ability to set a timeout on a request.
func NewDeleteClusterTemplateParamsWithTimeout(timeout time.Duration) *DeleteClusterTemplateParams {
	return &DeleteClusterTemplateParams{
		timeout: timeout,
	}
}

// NewDeleteClusterTemplateParamsWithContext creates a new DeleteClusterTemplateParams object
// with the ability to set a context for a request.
func NewDeleteClusterTemplateParamsWithContext(ctx context.Context) *DeleteClusterTemplateParams {
	return &DeleteClusterTemplateParams{
		Context: ctx,
	}
}

// NewDeleteClusterTemplateParamsWithHTTPClient creates a new DeleteClusterTemplateParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteClusterTemplateParamsWithHTTPClient(client *http.Client) *DeleteClusterTemplateParams {
	return &DeleteClusterTemplateParams{
		HTTPClient: client,
	}
}

/*
DeleteClusterTemplateParams contains all the parameters to send to the API endpoint

	for the delete cluster template operation.

	Typically these are written to a http.Request.
*/
type DeleteClusterTemplateParams struct {

	// ProjectID.
	ProjectID string

	// TemplateID.
	ClusterTemplateID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete cluster template params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteClusterTemplateParams) WithDefaults() *DeleteClusterTemplateParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete cluster template params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteClusterTemplateParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete cluster template params
func (o *DeleteClusterTemplateParams) WithTimeout(timeout time.Duration) *DeleteClusterTemplateParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete cluster template params
func (o *DeleteClusterTemplateParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete cluster template params
func (o *DeleteClusterTemplateParams) WithContext(ctx context.Context) *DeleteClusterTemplateParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete cluster template params
func (o *DeleteClusterTemplateParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete cluster template params
func (o *DeleteClusterTemplateParams) WithHTTPClient(client *http.Client) *DeleteClusterTemplateParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete cluster template params
func (o *DeleteClusterTemplateParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithProjectID adds the projectID to the delete cluster template params
func (o *DeleteClusterTemplateParams) WithProjectID(projectID string) *DeleteClusterTemplateParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the delete cluster template params
func (o *DeleteClusterTemplateParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithClusterTemplateID adds the templateID to the delete cluster template params
func (o *DeleteClusterTemplateParams) WithClusterTemplateID(templateID string) *DeleteClusterTemplateParams {
	o.SetClusterTemplateID(templateID)
	return o
}

// SetClusterTemplateID adds the templateId to the delete cluster template params
func (o *DeleteClusterTemplateParams) SetClusterTemplateID(templateID string) {
	o.ClusterTemplateID = templateID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteClusterTemplateParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	// path param template_id
	if err := r.SetPathParam("template_id", o.ClusterTemplateID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
