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

// NewListGCPDiskTypesNoCredentialsV2Params creates a new ListGCPDiskTypesNoCredentialsV2Params object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListGCPDiskTypesNoCredentialsV2Params() *ListGCPDiskTypesNoCredentialsV2Params {
	return &ListGCPDiskTypesNoCredentialsV2Params{
		timeout: cr.DefaultTimeout,
	}
}

// NewListGCPDiskTypesNoCredentialsV2ParamsWithTimeout creates a new ListGCPDiskTypesNoCredentialsV2Params object
// with the ability to set a timeout on a request.
func NewListGCPDiskTypesNoCredentialsV2ParamsWithTimeout(timeout time.Duration) *ListGCPDiskTypesNoCredentialsV2Params {
	return &ListGCPDiskTypesNoCredentialsV2Params{
		timeout: timeout,
	}
}

// NewListGCPDiskTypesNoCredentialsV2ParamsWithContext creates a new ListGCPDiskTypesNoCredentialsV2Params object
// with the ability to set a context for a request.
func NewListGCPDiskTypesNoCredentialsV2ParamsWithContext(ctx context.Context) *ListGCPDiskTypesNoCredentialsV2Params {
	return &ListGCPDiskTypesNoCredentialsV2Params{
		Context: ctx,
	}
}

// NewListGCPDiskTypesNoCredentialsV2ParamsWithHTTPClient creates a new ListGCPDiskTypesNoCredentialsV2Params object
// with the ability to set a custom HTTPClient for a request.
func NewListGCPDiskTypesNoCredentialsV2ParamsWithHTTPClient(client *http.Client) *ListGCPDiskTypesNoCredentialsV2Params {
	return &ListGCPDiskTypesNoCredentialsV2Params{
		HTTPClient: client,
	}
}

/*
ListGCPDiskTypesNoCredentialsV2Params contains all the parameters to send to the API endpoint

	for the list g c p disk types no credentials v2 operation.

	Typically these are written to a http.Request.
*/
type ListGCPDiskTypesNoCredentialsV2Params struct {

	// Zone.
	Zone *string

	// ClusterID.
	ClusterID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list g c p disk types no credentials v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListGCPDiskTypesNoCredentialsV2Params) WithDefaults() *ListGCPDiskTypesNoCredentialsV2Params {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list g c p disk types no credentials v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListGCPDiskTypesNoCredentialsV2Params) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) WithTimeout(timeout time.Duration) *ListGCPDiskTypesNoCredentialsV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) WithContext(ctx context.Context) *ListGCPDiskTypesNoCredentialsV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) WithHTTPClient(client *http.Client) *ListGCPDiskTypesNoCredentialsV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithZone adds the zone to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) WithZone(zone *string) *ListGCPDiskTypesNoCredentialsV2Params {
	o.SetZone(zone)
	return o
}

// SetZone adds the zone to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) SetZone(zone *string) {
	o.Zone = zone
}

// WithClusterID adds the clusterID to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) WithClusterID(clusterID string) *ListGCPDiskTypesNoCredentialsV2Params {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) WithProjectID(projectID string) *ListGCPDiskTypesNoCredentialsV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list g c p disk types no credentials v2 params
func (o *ListGCPDiskTypesNoCredentialsV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListGCPDiskTypesNoCredentialsV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Zone != nil {

		// header param Zone
		if err := r.SetHeaderParam("Zone", *o.Zone); err != nil {
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
