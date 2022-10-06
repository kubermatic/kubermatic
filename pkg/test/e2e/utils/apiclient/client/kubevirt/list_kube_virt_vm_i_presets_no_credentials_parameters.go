// Code generated by go-swagger; DO NOT EDIT.

package kubevirt

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

// NewListKubeVirtVMIPresetsNoCredentialsParams creates a new ListKubeVirtVMIPresetsNoCredentialsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListKubeVirtVMIPresetsNoCredentialsParams() *ListKubeVirtVMIPresetsNoCredentialsParams {
	return &ListKubeVirtVMIPresetsNoCredentialsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListKubeVirtVMIPresetsNoCredentialsParamsWithTimeout creates a new ListKubeVirtVMIPresetsNoCredentialsParams object
// with the ability to set a timeout on a request.
func NewListKubeVirtVMIPresetsNoCredentialsParamsWithTimeout(timeout time.Duration) *ListKubeVirtVMIPresetsNoCredentialsParams {
	return &ListKubeVirtVMIPresetsNoCredentialsParams{
		timeout: timeout,
	}
}

// NewListKubeVirtVMIPresetsNoCredentialsParamsWithContext creates a new ListKubeVirtVMIPresetsNoCredentialsParams object
// with the ability to set a context for a request.
func NewListKubeVirtVMIPresetsNoCredentialsParamsWithContext(ctx context.Context) *ListKubeVirtVMIPresetsNoCredentialsParams {
	return &ListKubeVirtVMIPresetsNoCredentialsParams{
		Context: ctx,
	}
}

// NewListKubeVirtVMIPresetsNoCredentialsParamsWithHTTPClient creates a new ListKubeVirtVMIPresetsNoCredentialsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListKubeVirtVMIPresetsNoCredentialsParamsWithHTTPClient(client *http.Client) *ListKubeVirtVMIPresetsNoCredentialsParams {
	return &ListKubeVirtVMIPresetsNoCredentialsParams{
		HTTPClient: client,
	}
}

/*
ListKubeVirtVMIPresetsNoCredentialsParams contains all the parameters to send to the API endpoint

	for the list kube virt VM i presets no credentials operation.

	Typically these are written to a http.Request.
*/
type ListKubeVirtVMIPresetsNoCredentialsParams struct {

	// DatacenterName.
	DatacenterName *string

	// ClusterID.
	ClusterID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list kube virt VM i presets no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) WithDefaults() *ListKubeVirtVMIPresetsNoCredentialsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list kube virt VM i presets no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) WithTimeout(timeout time.Duration) *ListKubeVirtVMIPresetsNoCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) WithContext(ctx context.Context) *ListKubeVirtVMIPresetsNoCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) WithHTTPClient(client *http.Client) *ListKubeVirtVMIPresetsNoCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithDatacenterName adds the datacenterName to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) WithDatacenterName(datacenterName *string) *ListKubeVirtVMIPresetsNoCredentialsParams {
	o.SetDatacenterName(datacenterName)
	return o
}

// SetDatacenterName adds the datacenterName to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) SetDatacenterName(datacenterName *string) {
	o.DatacenterName = datacenterName
}

// WithClusterID adds the clusterID to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) WithClusterID(clusterID string) *ListKubeVirtVMIPresetsNoCredentialsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) WithProjectID(projectID string) *ListKubeVirtVMIPresetsNoCredentialsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list kube virt VM i presets no credentials params
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListKubeVirtVMIPresetsNoCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.DatacenterName != nil {

		// header param DatacenterName
		if err := r.SetHeaderParam("DatacenterName", *o.DatacenterName); err != nil {
			return err
		}
	}

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
