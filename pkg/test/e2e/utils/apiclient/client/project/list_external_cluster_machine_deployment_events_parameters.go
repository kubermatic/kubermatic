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

// NewListExternalClusterMachineDeploymentEventsParams creates a new ListExternalClusterMachineDeploymentEventsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListExternalClusterMachineDeploymentEventsParams() *ListExternalClusterMachineDeploymentEventsParams {
	return &ListExternalClusterMachineDeploymentEventsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListExternalClusterMachineDeploymentEventsParamsWithTimeout creates a new ListExternalClusterMachineDeploymentEventsParams object
// with the ability to set a timeout on a request.
func NewListExternalClusterMachineDeploymentEventsParamsWithTimeout(timeout time.Duration) *ListExternalClusterMachineDeploymentEventsParams {
	return &ListExternalClusterMachineDeploymentEventsParams{
		timeout: timeout,
	}
}

// NewListExternalClusterMachineDeploymentEventsParamsWithContext creates a new ListExternalClusterMachineDeploymentEventsParams object
// with the ability to set a context for a request.
func NewListExternalClusterMachineDeploymentEventsParamsWithContext(ctx context.Context) *ListExternalClusterMachineDeploymentEventsParams {
	return &ListExternalClusterMachineDeploymentEventsParams{
		Context: ctx,
	}
}

// NewListExternalClusterMachineDeploymentEventsParamsWithHTTPClient creates a new ListExternalClusterMachineDeploymentEventsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListExternalClusterMachineDeploymentEventsParamsWithHTTPClient(client *http.Client) *ListExternalClusterMachineDeploymentEventsParams {
	return &ListExternalClusterMachineDeploymentEventsParams{
		HTTPClient: client,
	}
}

/* ListExternalClusterMachineDeploymentEventsParams contains all the parameters to send to the API endpoint
   for the list external cluster machine deployment events operation.

   Typically these are written to a http.Request.
*/
type ListExternalClusterMachineDeploymentEventsParams struct {

	// ClusterID.
	ClusterID string

	// MachinedeploymentID.
	MachineDeploymentID string

	// ProjectID.
	ProjectID string

	// Type.
	Type *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list external cluster machine deployment events params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListExternalClusterMachineDeploymentEventsParams) WithDefaults() *ListExternalClusterMachineDeploymentEventsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list external cluster machine deployment events params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListExternalClusterMachineDeploymentEventsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) WithTimeout(timeout time.Duration) *ListExternalClusterMachineDeploymentEventsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) WithContext(ctx context.Context) *ListExternalClusterMachineDeploymentEventsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) WithHTTPClient(client *http.Client) *ListExternalClusterMachineDeploymentEventsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) WithClusterID(clusterID string) *ListExternalClusterMachineDeploymentEventsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithMachineDeploymentID adds the machinedeploymentID to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) WithMachineDeploymentID(machinedeploymentID string) *ListExternalClusterMachineDeploymentEventsParams {
	o.SetMachineDeploymentID(machinedeploymentID)
	return o
}

// SetMachineDeploymentID adds the machinedeploymentId to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) SetMachineDeploymentID(machinedeploymentID string) {
	o.MachineDeploymentID = machinedeploymentID
}

// WithProjectID adds the projectID to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) WithProjectID(projectID string) *ListExternalClusterMachineDeploymentEventsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithType adds the typeVar to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) WithType(typeVar *string) *ListExternalClusterMachineDeploymentEventsParams {
	o.SetType(typeVar)
	return o
}

// SetType adds the type to the list external cluster machine deployment events params
func (o *ListExternalClusterMachineDeploymentEventsParams) SetType(typeVar *string) {
	o.Type = typeVar
}

// WriteToRequest writes these params to a swagger request
func (o *ListExternalClusterMachineDeploymentEventsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
