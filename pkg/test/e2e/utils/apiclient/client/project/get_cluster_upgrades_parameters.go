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

// NewGetClusterUpgradesParams creates a new GetClusterUpgradesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewGetClusterUpgradesParams() *GetClusterUpgradesParams {
	return &GetClusterUpgradesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewGetClusterUpgradesParamsWithTimeout creates a new GetClusterUpgradesParams object
// with the ability to set a timeout on a request.
func NewGetClusterUpgradesParamsWithTimeout(timeout time.Duration) *GetClusterUpgradesParams {
	return &GetClusterUpgradesParams{
		timeout: timeout,
	}
}

// NewGetClusterUpgradesParamsWithContext creates a new GetClusterUpgradesParams object
// with the ability to set a context for a request.
func NewGetClusterUpgradesParamsWithContext(ctx context.Context) *GetClusterUpgradesParams {
	return &GetClusterUpgradesParams{
		Context: ctx,
	}
}

// NewGetClusterUpgradesParamsWithHTTPClient creates a new GetClusterUpgradesParams object
// with the ability to set a custom HTTPClient for a request.
func NewGetClusterUpgradesParamsWithHTTPClient(client *http.Client) *GetClusterUpgradesParams {
	return &GetClusterUpgradesParams{
		HTTPClient: client,
	}
}

/*
GetClusterUpgradesParams contains all the parameters to send to the API endpoint

	for the get cluster upgrades operation.

	Typically these are written to a http.Request.
*/
type GetClusterUpgradesParams struct {

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

// WithDefaults hydrates default values in the get cluster upgrades params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetClusterUpgradesParams) WithDefaults() *GetClusterUpgradesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the get cluster upgrades params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *GetClusterUpgradesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the get cluster upgrades params
func (o *GetClusterUpgradesParams) WithTimeout(timeout time.Duration) *GetClusterUpgradesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get cluster upgrades params
func (o *GetClusterUpgradesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get cluster upgrades params
func (o *GetClusterUpgradesParams) WithContext(ctx context.Context) *GetClusterUpgradesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get cluster upgrades params
func (o *GetClusterUpgradesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get cluster upgrades params
func (o *GetClusterUpgradesParams) WithHTTPClient(client *http.Client) *GetClusterUpgradesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get cluster upgrades params
func (o *GetClusterUpgradesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the get cluster upgrades params
func (o *GetClusterUpgradesParams) WithClusterID(clusterID string) *GetClusterUpgradesParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get cluster upgrades params
func (o *GetClusterUpgradesParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the get cluster upgrades params
func (o *GetClusterUpgradesParams) WithDC(dc string) *GetClusterUpgradesParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the get cluster upgrades params
func (o *GetClusterUpgradesParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the get cluster upgrades params
func (o *GetClusterUpgradesParams) WithProjectID(projectID string) *GetClusterUpgradesParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get cluster upgrades params
func (o *GetClusterUpgradesParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *GetClusterUpgradesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
