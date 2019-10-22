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

	strfmt "github.com/go-openapi/strfmt"
)

// NewGetClusterRoleBindingParams creates a new GetClusterRoleBindingParams object
// with the default values initialized.
func NewGetClusterRoleBindingParams() *GetClusterRoleBindingParams {
	var ()
	return &GetClusterRoleBindingParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetClusterRoleBindingParamsWithTimeout creates a new GetClusterRoleBindingParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetClusterRoleBindingParamsWithTimeout(timeout time.Duration) *GetClusterRoleBindingParams {
	var ()
	return &GetClusterRoleBindingParams{

		timeout: timeout,
	}
}

// NewGetClusterRoleBindingParamsWithContext creates a new GetClusterRoleBindingParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetClusterRoleBindingParamsWithContext(ctx context.Context) *GetClusterRoleBindingParams {
	var ()
	return &GetClusterRoleBindingParams{

		Context: ctx,
	}
}

// NewGetClusterRoleBindingParamsWithHTTPClient creates a new GetClusterRoleBindingParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetClusterRoleBindingParamsWithHTTPClient(client *http.Client) *GetClusterRoleBindingParams {
	var ()
	return &GetClusterRoleBindingParams{
		HTTPClient: client,
	}
}

/*GetClusterRoleBindingParams contains all the parameters to send to the API endpoint
for the get cluster role binding operation typically these are written to a http.Request
*/
type GetClusterRoleBindingParams struct {

	/*BindingID*/
	BindingID string
	/*ClusterID*/
	ClusterID string
	/*Dc*/
	Dc string
	/*ProjectID*/
	ProjectID string
	/*RoleID*/
	RoleID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get cluster role binding params
func (o *GetClusterRoleBindingParams) WithTimeout(timeout time.Duration) *GetClusterRoleBindingParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get cluster role binding params
func (o *GetClusterRoleBindingParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get cluster role binding params
func (o *GetClusterRoleBindingParams) WithContext(ctx context.Context) *GetClusterRoleBindingParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get cluster role binding params
func (o *GetClusterRoleBindingParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get cluster role binding params
func (o *GetClusterRoleBindingParams) WithHTTPClient(client *http.Client) *GetClusterRoleBindingParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get cluster role binding params
func (o *GetClusterRoleBindingParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBindingID adds the bindingID to the get cluster role binding params
func (o *GetClusterRoleBindingParams) WithBindingID(bindingID string) *GetClusterRoleBindingParams {
	o.SetBindingID(bindingID)
	return o
}

// SetBindingID adds the bindingId to the get cluster role binding params
func (o *GetClusterRoleBindingParams) SetBindingID(bindingID string) {
	o.BindingID = bindingID
}

// WithClusterID adds the clusterID to the get cluster role binding params
func (o *GetClusterRoleBindingParams) WithClusterID(clusterID string) *GetClusterRoleBindingParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get cluster role binding params
func (o *GetClusterRoleBindingParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDc adds the dc to the get cluster role binding params
func (o *GetClusterRoleBindingParams) WithDc(dc string) *GetClusterRoleBindingParams {
	o.SetDc(dc)
	return o
}

// SetDc adds the dc to the get cluster role binding params
func (o *GetClusterRoleBindingParams) SetDc(dc string) {
	o.Dc = dc
}

// WithProjectID adds the projectID to the get cluster role binding params
func (o *GetClusterRoleBindingParams) WithProjectID(projectID string) *GetClusterRoleBindingParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get cluster role binding params
func (o *GetClusterRoleBindingParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithRoleID adds the roleID to the get cluster role binding params
func (o *GetClusterRoleBindingParams) WithRoleID(roleID string) *GetClusterRoleBindingParams {
	o.SetRoleID(roleID)
	return o
}

// SetRoleID adds the roleId to the get cluster role binding params
func (o *GetClusterRoleBindingParams) SetRoleID(roleID string) {
	o.RoleID = roleID
}

// WriteToRequest writes these params to a swagger request
func (o *GetClusterRoleBindingParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param binding_id
	if err := r.SetPathParam("binding_id", o.BindingID); err != nil {
		return err
	}

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
		return err
	}

	// path param dc
	if err := r.SetPathParam("dc", o.Dc); err != nil {
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
