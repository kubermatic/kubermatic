// Code generated by go-swagger; DO NOT EDIT.

package addon

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

// NewDeleteAddonV2Params creates a new DeleteAddonV2Params object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteAddonV2Params() *DeleteAddonV2Params {
	return &DeleteAddonV2Params{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteAddonV2ParamsWithTimeout creates a new DeleteAddonV2Params object
// with the ability to set a timeout on a request.
func NewDeleteAddonV2ParamsWithTimeout(timeout time.Duration) *DeleteAddonV2Params {
	return &DeleteAddonV2Params{
		timeout: timeout,
	}
}

// NewDeleteAddonV2ParamsWithContext creates a new DeleteAddonV2Params object
// with the ability to set a context for a request.
func NewDeleteAddonV2ParamsWithContext(ctx context.Context) *DeleteAddonV2Params {
	return &DeleteAddonV2Params{
		Context: ctx,
	}
}

// NewDeleteAddonV2ParamsWithHTTPClient creates a new DeleteAddonV2Params object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteAddonV2ParamsWithHTTPClient(client *http.Client) *DeleteAddonV2Params {
	return &DeleteAddonV2Params{
		HTTPClient: client,
	}
}

/*
DeleteAddonV2Params contains all the parameters to send to the API endpoint

	for the delete addon v2 operation.

	Typically these are written to a http.Request.
*/
type DeleteAddonV2Params struct {

	// AddonID.
	AddonID string

	// ClusterID.
	ClusterID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete addon v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteAddonV2Params) WithDefaults() *DeleteAddonV2Params {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete addon v2 params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteAddonV2Params) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete addon v2 params
func (o *DeleteAddonV2Params) WithTimeout(timeout time.Duration) *DeleteAddonV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete addon v2 params
func (o *DeleteAddonV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete addon v2 params
func (o *DeleteAddonV2Params) WithContext(ctx context.Context) *DeleteAddonV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete addon v2 params
func (o *DeleteAddonV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete addon v2 params
func (o *DeleteAddonV2Params) WithHTTPClient(client *http.Client) *DeleteAddonV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete addon v2 params
func (o *DeleteAddonV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAddonID adds the addonID to the delete addon v2 params
func (o *DeleteAddonV2Params) WithAddonID(addonID string) *DeleteAddonV2Params {
	o.SetAddonID(addonID)
	return o
}

// SetAddonID adds the addonId to the delete addon v2 params
func (o *DeleteAddonV2Params) SetAddonID(addonID string) {
	o.AddonID = addonID
}

// WithClusterID adds the clusterID to the delete addon v2 params
func (o *DeleteAddonV2Params) WithClusterID(clusterID string) *DeleteAddonV2Params {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the delete addon v2 params
func (o *DeleteAddonV2Params) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the delete addon v2 params
func (o *DeleteAddonV2Params) WithProjectID(projectID string) *DeleteAddonV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the delete addon v2 params
func (o *DeleteAddonV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteAddonV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param addon_id
	if err := r.SetPathParam("addon_id", o.AddonID); err != nil {
		return err
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
