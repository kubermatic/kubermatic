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

	strfmt "github.com/go-openapi/strfmt"
)

// NewListGCPSizesNoCredentialsParams creates a new ListGCPSizesNoCredentialsParams object
// with the default values initialized.
func NewListGCPSizesNoCredentialsParams() *ListGCPSizesNoCredentialsParams {
	var ()
	return &ListGCPSizesNoCredentialsParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewListGCPSizesNoCredentialsParamsWithTimeout creates a new ListGCPSizesNoCredentialsParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListGCPSizesNoCredentialsParamsWithTimeout(timeout time.Duration) *ListGCPSizesNoCredentialsParams {
	var ()
	return &ListGCPSizesNoCredentialsParams{

		timeout: timeout,
	}
}

// NewListGCPSizesNoCredentialsParamsWithContext creates a new ListGCPSizesNoCredentialsParams object
// with the default values initialized, and the ability to set a context for a request
func NewListGCPSizesNoCredentialsParamsWithContext(ctx context.Context) *ListGCPSizesNoCredentialsParams {
	var ()
	return &ListGCPSizesNoCredentialsParams{

		Context: ctx,
	}
}

// NewListGCPSizesNoCredentialsParamsWithHTTPClient creates a new ListGCPSizesNoCredentialsParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListGCPSizesNoCredentialsParamsWithHTTPClient(client *http.Client) *ListGCPSizesNoCredentialsParams {
	var ()
	return &ListGCPSizesNoCredentialsParams{
		HTTPClient: client,
	}
}

/*ListGCPSizesNoCredentialsParams contains all the parameters to send to the API endpoint
for the list g c p sizes no credentials operation typically these are written to a http.Request
*/
type ListGCPSizesNoCredentialsParams struct {

	/*Zone*/
	Zone *string
	/*ClusterID*/
	ClusterID string
	/*Dc*/
	Dc string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) WithTimeout(timeout time.Duration) *ListGCPSizesNoCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) WithContext(ctx context.Context) *ListGCPSizesNoCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) WithHTTPClient(client *http.Client) *ListGCPSizesNoCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithZone adds the zone to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) WithZone(zone *string) *ListGCPSizesNoCredentialsParams {
	o.SetZone(zone)
	return o
}

// SetZone adds the zone to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) SetZone(zone *string) {
	o.Zone = zone
}

// WithClusterID adds the clusterID to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) WithClusterID(clusterID string) *ListGCPSizesNoCredentialsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDc adds the dc to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) WithDc(dc string) *ListGCPSizesNoCredentialsParams {
	o.SetDc(dc)
	return o
}

// SetDc adds the dc to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) SetDc(dc string) {
	o.Dc = dc
}

// WithProjectID adds the projectID to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) WithProjectID(projectID string) *ListGCPSizesNoCredentialsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list g c p sizes no credentials params
func (o *ListGCPSizesNoCredentialsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListGCPSizesNoCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	// path param dc
	if err := r.SetPathParam("dc", o.Dc); err != nil {
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
