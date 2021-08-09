// Code generated by go-swagger; DO NOT EDIT.

package gcp

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

// NewListGCPZonesNoCredentialsParams creates a new ListGCPZonesNoCredentialsParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListGCPZonesNoCredentialsParams() *ListGCPZonesNoCredentialsParams {
	return &ListGCPZonesNoCredentialsParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListGCPZonesNoCredentialsParamsWithTimeout creates a new ListGCPZonesNoCredentialsParams object
// with the ability to set a timeout on a request.
func NewListGCPZonesNoCredentialsParamsWithTimeout(timeout time.Duration) *ListGCPZonesNoCredentialsParams {
	return &ListGCPZonesNoCredentialsParams{
		timeout: timeout,
	}
}

// NewListGCPZonesNoCredentialsParamsWithContext creates a new ListGCPZonesNoCredentialsParams object
// with the ability to set a context for a request.
func NewListGCPZonesNoCredentialsParamsWithContext(ctx context.Context) *ListGCPZonesNoCredentialsParams {
	return &ListGCPZonesNoCredentialsParams{
		Context: ctx,
	}
}

// NewListGCPZonesNoCredentialsParamsWithHTTPClient creates a new ListGCPZonesNoCredentialsParams object
// with the ability to set a custom HTTPClient for a request.
func NewListGCPZonesNoCredentialsParamsWithHTTPClient(client *http.Client) *ListGCPZonesNoCredentialsParams {
	return &ListGCPZonesNoCredentialsParams{
		HTTPClient: client,
	}
}

/* ListGCPZonesNoCredentialsParams contains all the parameters to send to the API endpoint
   for the list g c p zones no credentials operation.

   Typically these are written to a http.Request.
*/
type ListGCPZonesNoCredentialsParams struct {

	// ClusterID.
	ClusterID string

	// Dc.
	DC string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list g c p zones no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListGCPZonesNoCredentialsParams) WithDefaults() *ListGCPZonesNoCredentialsParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list g c p zones no credentials params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListGCPZonesNoCredentialsParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) WithTimeout(timeout time.Duration) *ListGCPZonesNoCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) WithContext(ctx context.Context) *ListGCPZonesNoCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) WithHTTPClient(client *http.Client) *ListGCPZonesNoCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) WithClusterID(clusterID string) *ListGCPZonesNoCredentialsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) WithDC(dc string) *ListGCPZonesNoCredentialsParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) WithProjectID(projectID string) *ListGCPZonesNoCredentialsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list g c p zones no credentials params
func (o *ListGCPZonesNoCredentialsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListGCPZonesNoCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
