// Code generated by go-swagger; DO NOT EDIT.

package nutanix

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

// NewListNutanixCategoriesNoCredentialsParams creates a new ListNutanixCategoriesNoCredentialsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListNutanixCategoriesNoCredentialsParams() *ListNutanixCategoriesNoCredentialsParams {
	return &ListNutanixCategoriesNoCredentialsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListNutanixCategoriesNoCredentialsParamsWithTimeout creates a new ListNutanixCategoriesNoCredentialsParams object
// with the ability to set a timeout on a request.
func NewListNutanixCategoriesNoCredentialsParamsWithTimeout(timeout time.Duration) *ListNutanixCategoriesNoCredentialsParams {
	return &ListNutanixCategoriesNoCredentialsParams{
		timeout: timeout,
	}
}

// NewListNutanixCategoriesNoCredentialsParamsWithContext creates a new ListNutanixCategoriesNoCredentialsParams object
// with the ability to set a context for a request.
func NewListNutanixCategoriesNoCredentialsParamsWithContext(ctx context.Context) *ListNutanixCategoriesNoCredentialsParams {
	return &ListNutanixCategoriesNoCredentialsParams{
		Context: ctx,
	}
}

// NewListNutanixCategoriesNoCredentialsParamsWithHTTPClient creates a new ListNutanixCategoriesNoCredentialsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListNutanixCategoriesNoCredentialsParamsWithHTTPClient(client *http.Client) *ListNutanixCategoriesNoCredentialsParams {
	return &ListNutanixCategoriesNoCredentialsParams{
		HTTPClient: client,
	}
}

/*
ListNutanixCategoriesNoCredentialsParams contains all the parameters to send to the API endpoint

	for the list nutanix categories no credentials operation.

	Typically these are written to a http.Request.
*/
type ListNutanixCategoriesNoCredentialsParams struct {

	// ClusterID.
	ClusterID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list nutanix categories no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListNutanixCategoriesNoCredentialsParams) WithDefaults() *ListNutanixCategoriesNoCredentialsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list nutanix categories no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListNutanixCategoriesNoCredentialsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list nutanix categories no credentials params
func (o *ListNutanixCategoriesNoCredentialsParams) WithTimeout(timeout time.Duration) *ListNutanixCategoriesNoCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list nutanix categories no credentials params
func (o *ListNutanixCategoriesNoCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list nutanix categories no credentials params
func (o *ListNutanixCategoriesNoCredentialsParams) WithContext(ctx context.Context) *ListNutanixCategoriesNoCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list nutanix categories no credentials params
func (o *ListNutanixCategoriesNoCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list nutanix categories no credentials params
func (o *ListNutanixCategoriesNoCredentialsParams) WithHTTPClient(client *http.Client) *ListNutanixCategoriesNoCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list nutanix categories no credentials params
func (o *ListNutanixCategoriesNoCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list nutanix categories no credentials params
func (o *ListNutanixCategoriesNoCredentialsParams) WithClusterID(clusterID string) *ListNutanixCategoriesNoCredentialsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list nutanix categories no credentials params
func (o *ListNutanixCategoriesNoCredentialsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the list nutanix categories no credentials params
func (o *ListNutanixCategoriesNoCredentialsParams) WithProjectID(projectID string) *ListNutanixCategoriesNoCredentialsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list nutanix categories no credentials params
func (o *ListNutanixCategoriesNoCredentialsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListNutanixCategoriesNoCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
