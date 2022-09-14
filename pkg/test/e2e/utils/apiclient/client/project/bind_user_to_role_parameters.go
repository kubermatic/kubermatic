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

// NewBindUserToRoleParams creates a new BindUserToRoleParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewBindUserToRoleParams() *BindUserToRoleParams {
	return &BindUserToRoleParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewBindUserToRoleParamsWithTimeout creates a new BindUserToRoleParams object
// with the ability to set a timeout on a request.
func NewBindUserToRoleParamsWithTimeout(timeout time.Duration) *BindUserToRoleParams {
	return &BindUserToRoleParams{
		timeout: timeout,
	}
}

// NewBindUserToRoleParamsWithContext creates a new BindUserToRoleParams object
// with the ability to set a context for a request.
func NewBindUserToRoleParamsWithContext(ctx context.Context) *BindUserToRoleParams {
	return &BindUserToRoleParams{
		Context: ctx,
	}
}

// NewBindUserToRoleParamsWithHTTPClient creates a new BindUserToRoleParams object
// with the ability to set a custom HTTPClient for a request.
func NewBindUserToRoleParamsWithHTTPClient(client *http.Client) *BindUserToRoleParams {
	return &BindUserToRoleParams{
		HTTPClient: client,
	}
}

/*
BindUserToRoleParams contains all the parameters to send to the API endpoint

	for the bind user to role operation.

	Typically these are written to a http.Request.
*/
type BindUserToRoleParams struct {

	// Body.
	Body *models.RoleUser

	// ClusterID.
	ClusterID string

	// Dc.
	DC string

	// Namespace.
	Namespace string

	// ProjectID.
	ProjectID string

	// RoleID.
	RoleID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the bind user to role params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *BindUserToRoleParams) WithDefaults() *BindUserToRoleParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the bind user to role params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *BindUserToRoleParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the bind user to role params
func (o *BindUserToRoleParams) WithTimeout(timeout time.Duration) *BindUserToRoleParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the bind user to role params
func (o *BindUserToRoleParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the bind user to role params
func (o *BindUserToRoleParams) WithContext(ctx context.Context) *BindUserToRoleParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the bind user to role params
func (o *BindUserToRoleParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the bind user to role params
func (o *BindUserToRoleParams) WithHTTPClient(client *http.Client) *BindUserToRoleParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the bind user to role params
func (o *BindUserToRoleParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the bind user to role params
func (o *BindUserToRoleParams) WithBody(body *models.RoleUser) *BindUserToRoleParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the bind user to role params
func (o *BindUserToRoleParams) SetBody(body *models.RoleUser) {
	o.Body = body
}

// WithClusterID adds the clusterID to the bind user to role params
func (o *BindUserToRoleParams) WithClusterID(clusterID string) *BindUserToRoleParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the bind user to role params
func (o *BindUserToRoleParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the bind user to role params
func (o *BindUserToRoleParams) WithDC(dc string) *BindUserToRoleParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the bind user to role params
func (o *BindUserToRoleParams) SetDC(dc string) {
	o.DC = dc
}

// WithNamespace adds the namespace to the bind user to role params
func (o *BindUserToRoleParams) WithNamespace(namespace string) *BindUserToRoleParams {
	o.SetNamespace(namespace)
	return o
}

// SetNamespace adds the namespace to the bind user to role params
func (o *BindUserToRoleParams) SetNamespace(namespace string) {
	o.Namespace = namespace
}

// WithProjectID adds the projectID to the bind user to role params
func (o *BindUserToRoleParams) WithProjectID(projectID string) *BindUserToRoleParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the bind user to role params
func (o *BindUserToRoleParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithRoleID adds the roleID to the bind user to role params
func (o *BindUserToRoleParams) WithRoleID(roleID string) *BindUserToRoleParams {
	o.SetRoleID(roleID)
	return o
}

// SetRoleID adds the roleId to the bind user to role params
func (o *BindUserToRoleParams) SetRoleID(roleID string) {
	o.RoleID = roleID
}

// WriteToRequest writes these params to a swagger request
func (o *BindUserToRoleParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
		return err
	}

	// path param dc
	if err := r.SetPathParam("dc", o.DC); err != nil {
		return err
	}

	// path param namespace
	if err := r.SetPathParam("namespace", o.Namespace); err != nil {
		return err
	}

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	// path param role_id
	if err := r.SetPathParam("role_id", o.RoleID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
