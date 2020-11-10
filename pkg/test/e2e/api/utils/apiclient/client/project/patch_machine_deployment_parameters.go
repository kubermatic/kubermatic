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

// NewPatchMachineDeploymentParams creates a new PatchMachineDeploymentParams object
// with the default values initialized.
func NewPatchMachineDeploymentParams() *PatchMachineDeploymentParams {
	var ()
	return &PatchMachineDeploymentParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewPatchMachineDeploymentParamsWithTimeout creates a new PatchMachineDeploymentParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewPatchMachineDeploymentParamsWithTimeout(timeout time.Duration) *PatchMachineDeploymentParams {
	var ()
	return &PatchMachineDeploymentParams{

		timeout: timeout,
	}
}

// NewPatchMachineDeploymentParamsWithContext creates a new PatchMachineDeploymentParams object
// with the default values initialized, and the ability to set a context for a request
func NewPatchMachineDeploymentParamsWithContext(ctx context.Context) *PatchMachineDeploymentParams {
	var ()
	return &PatchMachineDeploymentParams{

		Context: ctx,
	}
}

// NewPatchMachineDeploymentParamsWithHTTPClient creates a new PatchMachineDeploymentParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewPatchMachineDeploymentParamsWithHTTPClient(client *http.Client) *PatchMachineDeploymentParams {
	var ()
	return &PatchMachineDeploymentParams{
		HTTPClient: client,
	}
}

/*PatchMachineDeploymentParams contains all the parameters to send to the API endpoint
for the patch machine deployment operation typically these are written to a http.Request
*/
type PatchMachineDeploymentParams struct {

	/*Patch*/
	Patch interface{}
	/*ClusterID*/
	ClusterID string
	/*MachinedeploymentID*/
	MachineDeploymentID string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the patch machine deployment params
func (o *PatchMachineDeploymentParams) WithTimeout(timeout time.Duration) *PatchMachineDeploymentParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the patch machine deployment params
func (o *PatchMachineDeploymentParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the patch machine deployment params
func (o *PatchMachineDeploymentParams) WithContext(ctx context.Context) *PatchMachineDeploymentParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the patch machine deployment params
func (o *PatchMachineDeploymentParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the patch machine deployment params
func (o *PatchMachineDeploymentParams) WithHTTPClient(client *http.Client) *PatchMachineDeploymentParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the patch machine deployment params
func (o *PatchMachineDeploymentParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithPatch adds the patch to the patch machine deployment params
func (o *PatchMachineDeploymentParams) WithPatch(patch interface{}) *PatchMachineDeploymentParams {
	o.SetPatch(patch)
	return o
}

// SetPatch adds the patch to the patch machine deployment params
func (o *PatchMachineDeploymentParams) SetPatch(patch interface{}) {
	o.Patch = patch
}

// WithClusterID adds the clusterID to the patch machine deployment params
func (o *PatchMachineDeploymentParams) WithClusterID(clusterID string) *PatchMachineDeploymentParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the patch machine deployment params
func (o *PatchMachineDeploymentParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithMachineDeploymentID adds the machinedeploymentID to the patch machine deployment params
func (o *PatchMachineDeploymentParams) WithMachineDeploymentID(machinedeploymentID string) *PatchMachineDeploymentParams {
	o.SetMachineDeploymentID(machinedeploymentID)
	return o
}

// SetMachineDeploymentID adds the machinedeploymentId to the patch machine deployment params
func (o *PatchMachineDeploymentParams) SetMachineDeploymentID(machinedeploymentID string) {
	o.MachineDeploymentID = machinedeploymentID
}

// WithProjectID adds the projectID to the patch machine deployment params
func (o *PatchMachineDeploymentParams) WithProjectID(projectID string) *PatchMachineDeploymentParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the patch machine deployment params
func (o *PatchMachineDeploymentParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *PatchMachineDeploymentParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	// path param machinedeployment_id
	if err := r.SetPathParam("machinedeployment_id", o.MachineDeploymentID); err != nil {
		return err
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
