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

// NewGetExternalClusterNodeParams creates a new GetExternalClusterNodeParams object
// with the default values initialized.
func NewGetExternalClusterNodeParams() *GetExternalClusterNodeParams {
	var ()
	return &GetExternalClusterNodeParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetExternalClusterNodeParamsWithTimeout creates a new GetExternalClusterNodeParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetExternalClusterNodeParamsWithTimeout(timeout time.Duration) *GetExternalClusterNodeParams {
	var ()
	return &GetExternalClusterNodeParams{

		timeout: timeout,
	}
}

// NewGetExternalClusterNodeParamsWithContext creates a new GetExternalClusterNodeParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetExternalClusterNodeParamsWithContext(ctx context.Context) *GetExternalClusterNodeParams {
	var ()
	return &GetExternalClusterNodeParams{

		Context: ctx,
	}
}

// NewGetExternalClusterNodeParamsWithHTTPClient creates a new GetExternalClusterNodeParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetExternalClusterNodeParamsWithHTTPClient(client *http.Client) *GetExternalClusterNodeParams {
	var ()
	return &GetExternalClusterNodeParams{
		HTTPClient: client,
	}
}

/*GetExternalClusterNodeParams contains all the parameters to send to the API endpoint
for the get external cluster node operation typically these are written to a http.Request
*/
type GetExternalClusterNodeParams struct {

	/*ClusterID*/
	ClusterID string
	/*NodeID*/
	NodeID string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get external cluster node params
func (o *GetExternalClusterNodeParams) WithTimeout(timeout time.Duration) *GetExternalClusterNodeParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get external cluster node params
func (o *GetExternalClusterNodeParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get external cluster node params
func (o *GetExternalClusterNodeParams) WithContext(ctx context.Context) *GetExternalClusterNodeParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get external cluster node params
func (o *GetExternalClusterNodeParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get external cluster node params
func (o *GetExternalClusterNodeParams) WithHTTPClient(client *http.Client) *GetExternalClusterNodeParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get external cluster node params
func (o *GetExternalClusterNodeParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the get external cluster node params
func (o *GetExternalClusterNodeParams) WithClusterID(clusterID string) *GetExternalClusterNodeParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get external cluster node params
func (o *GetExternalClusterNodeParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithNodeID adds the nodeID to the get external cluster node params
func (o *GetExternalClusterNodeParams) WithNodeID(nodeID string) *GetExternalClusterNodeParams {
	o.SetNodeID(nodeID)
	return o
}

// SetNodeID adds the nodeId to the get external cluster node params
func (o *GetExternalClusterNodeParams) SetNodeID(nodeID string) {
	o.NodeID = nodeID
}

// WithProjectID adds the projectID to the get external cluster node params
func (o *GetExternalClusterNodeParams) WithProjectID(projectID string) *GetExternalClusterNodeParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get external cluster node params
func (o *GetExternalClusterNodeParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *GetExternalClusterNodeParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
		return err
	}

	// path param node_id
	if err := r.SetPathParam("node_id", o.NodeID); err != nil {
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
