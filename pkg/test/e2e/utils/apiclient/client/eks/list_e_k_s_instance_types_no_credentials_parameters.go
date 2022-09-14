// Code generated by go-swagger; DO NOT EDIT.

package eks

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

// NewListEKSInstanceTypesNoCredentialsParams creates a new ListEKSInstanceTypesNoCredentialsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListEKSInstanceTypesNoCredentialsParams() *ListEKSInstanceTypesNoCredentialsParams {
	return &ListEKSInstanceTypesNoCredentialsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListEKSInstanceTypesNoCredentialsParamsWithTimeout creates a new ListEKSInstanceTypesNoCredentialsParams object
// with the ability to set a timeout on a request.
func NewListEKSInstanceTypesNoCredentialsParamsWithTimeout(timeout time.Duration) *ListEKSInstanceTypesNoCredentialsParams {
	return &ListEKSInstanceTypesNoCredentialsParams{
		timeout: timeout,
	}
}

// NewListEKSInstanceTypesNoCredentialsParamsWithContext creates a new ListEKSInstanceTypesNoCredentialsParams object
// with the ability to set a context for a request.
func NewListEKSInstanceTypesNoCredentialsParamsWithContext(ctx context.Context) *ListEKSInstanceTypesNoCredentialsParams {
	return &ListEKSInstanceTypesNoCredentialsParams{
		Context: ctx,
	}
}

// NewListEKSInstanceTypesNoCredentialsParamsWithHTTPClient creates a new ListEKSInstanceTypesNoCredentialsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListEKSInstanceTypesNoCredentialsParamsWithHTTPClient(client *http.Client) *ListEKSInstanceTypesNoCredentialsParams {
	return &ListEKSInstanceTypesNoCredentialsParams{
		HTTPClient: client,
	}
}

/*
ListEKSInstanceTypesNoCredentialsParams contains all the parameters to send to the API endpoint

	for the list e k s instance types no credentials operation.

	Typically these are written to a http.Request.
*/
type ListEKSInstanceTypesNoCredentialsParams struct {

	// ClusterID.
	ClusterID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list e k s instance types no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListEKSInstanceTypesNoCredentialsParams) WithDefaults() *ListEKSInstanceTypesNoCredentialsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list e k s instance types no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListEKSInstanceTypesNoCredentialsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list e k s instance types no credentials params
func (o *ListEKSInstanceTypesNoCredentialsParams) WithTimeout(timeout time.Duration) *ListEKSInstanceTypesNoCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list e k s instance types no credentials params
func (o *ListEKSInstanceTypesNoCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list e k s instance types no credentials params
func (o *ListEKSInstanceTypesNoCredentialsParams) WithContext(ctx context.Context) *ListEKSInstanceTypesNoCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list e k s instance types no credentials params
func (o *ListEKSInstanceTypesNoCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list e k s instance types no credentials params
func (o *ListEKSInstanceTypesNoCredentialsParams) WithHTTPClient(client *http.Client) *ListEKSInstanceTypesNoCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list e k s instance types no credentials params
func (o *ListEKSInstanceTypesNoCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list e k s instance types no credentials params
func (o *ListEKSInstanceTypesNoCredentialsParams) WithClusterID(clusterID string) *ListEKSInstanceTypesNoCredentialsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list e k s instance types no credentials params
func (o *ListEKSInstanceTypesNoCredentialsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the list e k s instance types no credentials params
func (o *ListEKSInstanceTypesNoCredentialsParams) WithProjectID(projectID string) *ListEKSInstanceTypesNoCredentialsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list e k s instance types no credentials params
func (o *ListEKSInstanceTypesNoCredentialsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListEKSInstanceTypesNoCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
