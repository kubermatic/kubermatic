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

// NewGetMachineDeploymentParams creates a new GetMachineDeploymentParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetMachineDeploymentParams() *GetMachineDeploymentParams {
	return &GetMachineDeploymentParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetMachineDeploymentParamsWithTimeout creates a new GetMachineDeploymentParams object
// with the ability to set a timeout on a request.
func NewGetMachineDeploymentParamsWithTimeout(timeout time.Duration) *GetMachineDeploymentParams {
	return &GetMachineDeploymentParams{
		timeout: timeout,
	}
}

// NewGetMachineDeploymentParamsWithContext creates a new GetMachineDeploymentParams object
// with the ability to set a context for a request.
func NewGetMachineDeploymentParamsWithContext(ctx context.Context) *GetMachineDeploymentParams {
	return &GetMachineDeploymentParams{
		Context: ctx,
	}
}

// NewGetMachineDeploymentParamsWithHTTPClient creates a new GetMachineDeploymentParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetMachineDeploymentParamsWithHTTPClient(client *http.Client) *GetMachineDeploymentParams {
	return &GetMachineDeploymentParams{
		HTTPClient: client,
	}
}

/*
GetMachineDeploymentParams contains all the parameters to send to the API endpoint

	for the get machine deployment operation.

	Typically these are written to a http.Request.
*/
type GetMachineDeploymentParams struct {

	// ClusterID.
	ClusterID string

	// MachinedeploymentID.
	MachineDeploymentID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get machine deployment params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetMachineDeploymentParams) WithDefaults() *GetMachineDeploymentParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get machine deployment params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetMachineDeploymentParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get machine deployment params
func (o *GetMachineDeploymentParams) WithTimeout(timeout time.Duration) *GetMachineDeploymentParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get machine deployment params
func (o *GetMachineDeploymentParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get machine deployment params
func (o *GetMachineDeploymentParams) WithContext(ctx context.Context) *GetMachineDeploymentParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get machine deployment params
func (o *GetMachineDeploymentParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get machine deployment params
func (o *GetMachineDeploymentParams) WithHTTPClient(client *http.Client) *GetMachineDeploymentParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get machine deployment params
func (o *GetMachineDeploymentParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the get machine deployment params
func (o *GetMachineDeploymentParams) WithClusterID(clusterID string) *GetMachineDeploymentParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get machine deployment params
func (o *GetMachineDeploymentParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithMachineDeploymentID adds the machinedeploymentID to the get machine deployment params
func (o *GetMachineDeploymentParams) WithMachineDeploymentID(machinedeploymentID string) *GetMachineDeploymentParams {
	o.SetMachineDeploymentID(machinedeploymentID)
	return o
}

// SetMachineDeploymentID adds the machinedeploymentId to the get machine deployment params
func (o *GetMachineDeploymentParams) SetMachineDeploymentID(machinedeploymentID string) {
	o.MachineDeploymentID = machinedeploymentID
}

// WithProjectID adds the projectID to the get machine deployment params
func (o *GetMachineDeploymentParams) WithProjectID(projectID string) *GetMachineDeploymentParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get machine deployment params
func (o *GetMachineDeploymentParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *GetMachineDeploymentParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

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
