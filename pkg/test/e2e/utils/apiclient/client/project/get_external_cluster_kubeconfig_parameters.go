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

// NewGetExternalClusterKubeconfigParams creates a new GetExternalClusterKubeconfigParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetExternalClusterKubeconfigParams() *GetExternalClusterKubeconfigParams {
	return &GetExternalClusterKubeconfigParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetExternalClusterKubeconfigParamsWithTimeout creates a new GetExternalClusterKubeconfigParams object
// with the ability to set a timeout on a request.
func NewGetExternalClusterKubeconfigParamsWithTimeout(timeout time.Duration) *GetExternalClusterKubeconfigParams {
	return &GetExternalClusterKubeconfigParams{
		timeout: timeout,
	}
}

// NewGetExternalClusterKubeconfigParamsWithContext creates a new GetExternalClusterKubeconfigParams object
// with the ability to set a context for a request.
func NewGetExternalClusterKubeconfigParamsWithContext(ctx context.Context) *GetExternalClusterKubeconfigParams {
	return &GetExternalClusterKubeconfigParams{
		Context: ctx,
	}
}

// NewGetExternalClusterKubeconfigParamsWithHTTPClient creates a new GetExternalClusterKubeconfigParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetExternalClusterKubeconfigParamsWithHTTPClient(client *http.Client) *GetExternalClusterKubeconfigParams {
	return &GetExternalClusterKubeconfigParams{
		HTTPClient: client,
	}
}

/*
GetExternalClusterKubeconfigParams contains all the parameters to send to the API endpoint

	for the get external cluster kubeconfig operation.

	Typically these are written to a http.Request.
*/
type GetExternalClusterKubeconfigParams struct {

	// ClusterID.
	ClusterID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the get external cluster kubeconfig params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetExternalClusterKubeconfigParams) WithDefaults() *GetExternalClusterKubeconfigParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get external cluster kubeconfig params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetExternalClusterKubeconfigParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get external cluster kubeconfig params
func (o *GetExternalClusterKubeconfigParams) WithTimeout(timeout time.Duration) *GetExternalClusterKubeconfigParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get external cluster kubeconfig params
func (o *GetExternalClusterKubeconfigParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get external cluster kubeconfig params
func (o *GetExternalClusterKubeconfigParams) WithContext(ctx context.Context) *GetExternalClusterKubeconfigParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get external cluster kubeconfig params
func (o *GetExternalClusterKubeconfigParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get external cluster kubeconfig params
func (o *GetExternalClusterKubeconfigParams) WithHTTPClient(client *http.Client) *GetExternalClusterKubeconfigParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get external cluster kubeconfig params
func (o *GetExternalClusterKubeconfigParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the get external cluster kubeconfig params
func (o *GetExternalClusterKubeconfigParams) WithClusterID(clusterID string) *GetExternalClusterKubeconfigParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get external cluster kubeconfig params
func (o *GetExternalClusterKubeconfigParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the get external cluster kubeconfig params
func (o *GetExternalClusterKubeconfigParams) WithProjectID(projectID string) *GetExternalClusterKubeconfigParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get external cluster kubeconfig params
func (o *GetExternalClusterKubeconfigParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *GetExternalClusterKubeconfigParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
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
