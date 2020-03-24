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

// NewGetOidcClusterKubeconfigParams creates a new GetOidcClusterKubeconfigParams object
// with the default values initialized.
func NewGetOidcClusterKubeconfigParams() *GetOidcClusterKubeconfigParams {
	var ()
	return &GetOidcClusterKubeconfigParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetOidcClusterKubeconfigParamsWithTimeout creates a new GetOidcClusterKubeconfigParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetOidcClusterKubeconfigParamsWithTimeout(timeout time.Duration) *GetOidcClusterKubeconfigParams {
	var ()
	return &GetOidcClusterKubeconfigParams{

		timeout: timeout,
	}
}

// NewGetOidcClusterKubeconfigParamsWithContext creates a new GetOidcClusterKubeconfigParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetOidcClusterKubeconfigParamsWithContext(ctx context.Context) *GetOidcClusterKubeconfigParams {
	var ()
	return &GetOidcClusterKubeconfigParams{

		Context: ctx,
	}
}

// NewGetOidcClusterKubeconfigParamsWithHTTPClient creates a new GetOidcClusterKubeconfigParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetOidcClusterKubeconfigParamsWithHTTPClient(client *http.Client) *GetOidcClusterKubeconfigParams {
	var ()
	return &GetOidcClusterKubeconfigParams{
		HTTPClient: client,
	}
}

/*GetOidcClusterKubeconfigParams contains all the parameters to send to the API endpoint
for the get oidc cluster kubeconfig operation typically these are written to a http.Request
*/
type GetOidcClusterKubeconfigParams struct {

	/*ClusterID*/
	ClusterID string
	/*Dc*/
	DC string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) WithTimeout(timeout time.Duration) *GetOidcClusterKubeconfigParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) WithContext(ctx context.Context) *GetOidcClusterKubeconfigParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) WithHTTPClient(client *http.Client) *GetOidcClusterKubeconfigParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) WithClusterID(clusterID string) *GetOidcClusterKubeconfigParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) WithDC(dc string) *GetOidcClusterKubeconfigParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) WithProjectID(projectID string) *GetOidcClusterKubeconfigParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get oidc cluster kubeconfig params
func (o *GetOidcClusterKubeconfigParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *GetOidcClusterKubeconfigParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
