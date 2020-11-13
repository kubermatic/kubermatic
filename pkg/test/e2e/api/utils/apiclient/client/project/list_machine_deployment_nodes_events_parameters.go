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

// NewListMachineDeploymentNodesEventsParams creates a new ListMachineDeploymentNodesEventsParams object
// with the default values initialized.
func NewListMachineDeploymentNodesEventsParams() *ListMachineDeploymentNodesEventsParams {
	var ()
	return &ListMachineDeploymentNodesEventsParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewListMachineDeploymentNodesEventsParamsWithTimeout creates a new ListMachineDeploymentNodesEventsParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListMachineDeploymentNodesEventsParamsWithTimeout(timeout time.Duration) *ListMachineDeploymentNodesEventsParams {
	var ()
	return &ListMachineDeploymentNodesEventsParams{

		timeout: timeout,
	}
}

// NewListMachineDeploymentNodesEventsParamsWithContext creates a new ListMachineDeploymentNodesEventsParams object
// with the default values initialized, and the ability to set a context for a request
func NewListMachineDeploymentNodesEventsParamsWithContext(ctx context.Context) *ListMachineDeploymentNodesEventsParams {
	var ()
	return &ListMachineDeploymentNodesEventsParams{

		Context: ctx,
	}
}

// NewListMachineDeploymentNodesEventsParamsWithHTTPClient creates a new ListMachineDeploymentNodesEventsParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListMachineDeploymentNodesEventsParamsWithHTTPClient(client *http.Client) *ListMachineDeploymentNodesEventsParams {
	var ()
	return &ListMachineDeploymentNodesEventsParams{
		HTTPClient: client,
	}
}

/*ListMachineDeploymentNodesEventsParams contains all the parameters to send to the API endpoint
for the list machine deployment nodes events operation typically these are written to a http.Request
*/
type ListMachineDeploymentNodesEventsParams struct {

	/*ClusterID*/
	ClusterID string
	/*MachinedeploymentID*/
	MachineDeploymentID string
	/*ProjectID*/
	ProjectID string
	/*Type*/
	Type *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) WithTimeout(timeout time.Duration) *ListMachineDeploymentNodesEventsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) WithContext(ctx context.Context) *ListMachineDeploymentNodesEventsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) WithHTTPClient(client *http.Client) *ListMachineDeploymentNodesEventsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) WithClusterID(clusterID string) *ListMachineDeploymentNodesEventsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithMachineDeploymentID adds the machinedeploymentID to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) WithMachineDeploymentID(machinedeploymentID string) *ListMachineDeploymentNodesEventsParams {
	o.SetMachineDeploymentID(machinedeploymentID)
	return o
}

// SetMachineDeploymentID adds the machinedeploymentId to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) SetMachineDeploymentID(machinedeploymentID string) {
	o.MachineDeploymentID = machinedeploymentID
}

// WithProjectID adds the projectID to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) WithProjectID(projectID string) *ListMachineDeploymentNodesEventsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithType adds the typeVar to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) WithType(typeVar *string) *ListMachineDeploymentNodesEventsParams {
	o.SetType(typeVar)
	return o
}

// SetType adds the type to the list machine deployment nodes events params
func (o *ListMachineDeploymentNodesEventsParams) SetType(typeVar *string) {
	o.Type = typeVar
}

// WriteToRequest writes these params to a swagger request
func (o *ListMachineDeploymentNodesEventsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if o.Type != nil {

		// query param type
		var qrType string
		if o.Type != nil {
			qrType = *o.Type
		}
		qType := qrType
		if qType != "" {
			if err := r.SetQueryParam("type", qType); err != nil {
				return err
			}
		}

	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
