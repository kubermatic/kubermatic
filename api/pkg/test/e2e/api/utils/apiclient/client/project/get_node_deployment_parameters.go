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

// NewGetNodeDeploymentParams creates a new GetNodeDeploymentParams object
// with the default values initialized.
func NewGetNodeDeploymentParams() *GetNodeDeploymentParams {
	var ()
	return &GetNodeDeploymentParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetNodeDeploymentParamsWithTimeout creates a new GetNodeDeploymentParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetNodeDeploymentParamsWithTimeout(timeout time.Duration) *GetNodeDeploymentParams {
	var ()
	return &GetNodeDeploymentParams{

		timeout: timeout,
	}
}

// NewGetNodeDeploymentParamsWithContext creates a new GetNodeDeploymentParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetNodeDeploymentParamsWithContext(ctx context.Context) *GetNodeDeploymentParams {
	var ()
	return &GetNodeDeploymentParams{

		Context: ctx,
	}
}

// NewGetNodeDeploymentParamsWithHTTPClient creates a new GetNodeDeploymentParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetNodeDeploymentParamsWithHTTPClient(client *http.Client) *GetNodeDeploymentParams {
	var ()
	return &GetNodeDeploymentParams{
		HTTPClient: client,
	}
}

/*GetNodeDeploymentParams contains all the parameters to send to the API endpoint
for the get node deployment operation typically these are written to a http.Request
*/
type GetNodeDeploymentParams struct {

	/*ClusterID*/
	ClusterID string
	/*Dc*/
	DC string
	/*NodedeploymentID*/
	NodeDeploymentID string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get node deployment params
func (o *GetNodeDeploymentParams) WithTimeout(timeout time.Duration) *GetNodeDeploymentParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get node deployment params
func (o *GetNodeDeploymentParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get node deployment params
func (o *GetNodeDeploymentParams) WithContext(ctx context.Context) *GetNodeDeploymentParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get node deployment params
func (o *GetNodeDeploymentParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get node deployment params
func (o *GetNodeDeploymentParams) WithHTTPClient(client *http.Client) *GetNodeDeploymentParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get node deployment params
func (o *GetNodeDeploymentParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the get node deployment params
func (o *GetNodeDeploymentParams) WithClusterID(clusterID string) *GetNodeDeploymentParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get node deployment params
func (o *GetNodeDeploymentParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the get node deployment params
func (o *GetNodeDeploymentParams) WithDC(dc string) *GetNodeDeploymentParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the get node deployment params
func (o *GetNodeDeploymentParams) SetDC(dc string) {
	o.DC = dc
}

// WithNodeDeploymentID adds the nodedeploymentID to the get node deployment params
func (o *GetNodeDeploymentParams) WithNodeDeploymentID(nodedeploymentID string) *GetNodeDeploymentParams {
	o.SetNodeDeploymentID(nodedeploymentID)
	return o
}

// SetNodeDeploymentID adds the nodedeploymentId to the get node deployment params
func (o *GetNodeDeploymentParams) SetNodeDeploymentID(nodedeploymentID string) {
	o.NodeDeploymentID = nodedeploymentID
}

// WithProjectID adds the projectID to the get node deployment params
func (o *GetNodeDeploymentParams) WithProjectID(projectID string) *GetNodeDeploymentParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get node deployment params
func (o *GetNodeDeploymentParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *GetNodeDeploymentParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
		return err
	}

	// path param dc
	if err := r.SetPathParam("dc", o.DC); err != nil {
		return err
	}

	// path param nodedeployment_id
	if err := r.SetPathParam("nodedeployment_id", o.NodeDeploymentID); err != nil {
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
