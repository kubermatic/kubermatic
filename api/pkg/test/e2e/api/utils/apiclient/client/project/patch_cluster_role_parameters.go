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

// NewPatchClusterRoleParams creates a new PatchClusterRoleParams object
// with the default values initialized.
func NewPatchClusterRoleParams() *PatchClusterRoleParams {
	var ()
	return &PatchClusterRoleParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewPatchClusterRoleParamsWithTimeout creates a new PatchClusterRoleParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewPatchClusterRoleParamsWithTimeout(timeout time.Duration) *PatchClusterRoleParams {
	var ()
	return &PatchClusterRoleParams{

		timeout: timeout,
	}
}

// NewPatchClusterRoleParamsWithContext creates a new PatchClusterRoleParams object
// with the default values initialized, and the ability to set a context for a request
func NewPatchClusterRoleParamsWithContext(ctx context.Context) *PatchClusterRoleParams {
	var ()
	return &PatchClusterRoleParams{

		Context: ctx,
	}
}

// NewPatchClusterRoleParamsWithHTTPClient creates a new PatchClusterRoleParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewPatchClusterRoleParamsWithHTTPClient(client *http.Client) *PatchClusterRoleParams {
	var ()
	return &PatchClusterRoleParams{
		HTTPClient: client,
	}
}

/*PatchClusterRoleParams contains all the parameters to send to the API endpoint
for the patch cluster role operation typically these are written to a http.Request
*/
type PatchClusterRoleParams struct {

	/*Patch*/
	Patch interface{}
	/*ClusterID*/
	ClusterID string
	/*Dc*/
	DC string
	/*ProjectID*/
	ProjectID string
	/*RoleID*/
	RoleID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the patch cluster role params
func (o *PatchClusterRoleParams) WithTimeout(timeout time.Duration) *PatchClusterRoleParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the patch cluster role params
func (o *PatchClusterRoleParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the patch cluster role params
func (o *PatchClusterRoleParams) WithContext(ctx context.Context) *PatchClusterRoleParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the patch cluster role params
func (o *PatchClusterRoleParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the patch cluster role params
func (o *PatchClusterRoleParams) WithHTTPClient(client *http.Client) *PatchClusterRoleParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the patch cluster role params
func (o *PatchClusterRoleParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithPatch adds the patch to the patch cluster role params
func (o *PatchClusterRoleParams) WithPatch(patch interface{}) *PatchClusterRoleParams {
	o.SetPatch(patch)
	return o
}

// SetPatch adds the patch to the patch cluster role params
func (o *PatchClusterRoleParams) SetPatch(patch interface{}) {
	o.Patch = patch
}

// WithClusterID adds the clusterID to the patch cluster role params
func (o *PatchClusterRoleParams) WithClusterID(clusterID string) *PatchClusterRoleParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the patch cluster role params
func (o *PatchClusterRoleParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the patch cluster role params
func (o *PatchClusterRoleParams) WithDC(dc string) *PatchClusterRoleParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the patch cluster role params
func (o *PatchClusterRoleParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the patch cluster role params
func (o *PatchClusterRoleParams) WithProjectID(projectID string) *PatchClusterRoleParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the patch cluster role params
func (o *PatchClusterRoleParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithRoleID adds the roleID to the patch cluster role params
func (o *PatchClusterRoleParams) WithRoleID(roleID string) *PatchClusterRoleParams {
	o.SetRoleID(roleID)
	return o
}

// SetRoleID adds the roleId to the patch cluster role params
func (o *PatchClusterRoleParams) SetRoleID(roleID string) {
	o.RoleID = roleID
}

// WriteToRequest writes these params to a swagger request
func (o *PatchClusterRoleParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Patch != nil {
		if err := r.SetBodyParam(o.Patch); err != nil {
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
