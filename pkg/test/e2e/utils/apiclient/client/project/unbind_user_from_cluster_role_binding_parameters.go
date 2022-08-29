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

// NewUnbindUserFromClusterRoleBindingParams creates a new UnbindUserFromClusterRoleBindingParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewUnbindUserFromClusterRoleBindingParams() *UnbindUserFromClusterRoleBindingParams {
	return &UnbindUserFromClusterRoleBindingParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewUnbindUserFromClusterRoleBindingParamsWithTimeout creates a new UnbindUserFromClusterRoleBindingParams object
// with the ability to set a timeout on a request.
func NewUnbindUserFromClusterRoleBindingParamsWithTimeout(timeout time.Duration) *UnbindUserFromClusterRoleBindingParams {
	return &UnbindUserFromClusterRoleBindingParams{
		timeout: timeout,
	}
}

// NewUnbindUserFromClusterRoleBindingParamsWithContext creates a new UnbindUserFromClusterRoleBindingParams object
// with the ability to set a context for a request.
func NewUnbindUserFromClusterRoleBindingParamsWithContext(ctx context.Context) *UnbindUserFromClusterRoleBindingParams {
	return &UnbindUserFromClusterRoleBindingParams{
		Context: ctx,
	}
}

// NewUnbindUserFromClusterRoleBindingParamsWithHTTPClient creates a new UnbindUserFromClusterRoleBindingParams object
// with the ability to set a custom HTTPClient for a request.
func NewUnbindUserFromClusterRoleBindingParamsWithHTTPClient(client *http.Client) *UnbindUserFromClusterRoleBindingParams {
	return &UnbindUserFromClusterRoleBindingParams{
		HTTPClient: client,
	}
}

/*
UnbindUserFromClusterRoleBindingParams contains all the parameters to send to the API endpoint

	for the unbind user from cluster role binding operation.

	Typically these are written to a http.Request.
*/
type UnbindUserFromClusterRoleBindingParams struct {

	// Body.
	Body *models.ClusterRoleUser

	// ClusterID.
	ClusterID string

	// Dc.
	DC string

	// ProjectID.
	ProjectID string

	// RoleID.
	RoleID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the unbind user from cluster role binding params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UnbindUserFromClusterRoleBindingParams) WithDefaults() *UnbindUserFromClusterRoleBindingParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the unbind user from cluster role binding params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UnbindUserFromClusterRoleBindingParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) WithTimeout(timeout time.Duration) *UnbindUserFromClusterRoleBindingParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) WithContext(ctx context.Context) *UnbindUserFromClusterRoleBindingParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) WithHTTPClient(client *http.Client) *UnbindUserFromClusterRoleBindingParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) WithBody(body *models.ClusterRoleUser) *UnbindUserFromClusterRoleBindingParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) SetBody(body *models.ClusterRoleUser) {
	o.Body = body
}

// WithClusterID adds the clusterID to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) WithClusterID(clusterID string) *UnbindUserFromClusterRoleBindingParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) WithDC(dc string) *UnbindUserFromClusterRoleBindingParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) WithProjectID(projectID string) *UnbindUserFromClusterRoleBindingParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithRoleID adds the roleID to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) WithRoleID(roleID string) *UnbindUserFromClusterRoleBindingParams {
	o.SetRoleID(roleID)
	return o
}

// SetRoleID adds the roleId to the unbind user from cluster role binding params
func (o *UnbindUserFromClusterRoleBindingParams) SetRoleID(roleID string) {
	o.RoleID = roleID
}

// WriteToRequest writes these params to a swagger request
func (o *UnbindUserFromClusterRoleBindingParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
