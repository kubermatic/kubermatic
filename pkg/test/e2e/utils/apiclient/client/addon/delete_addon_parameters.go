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

// NewDeleteAddonParams creates a new DeleteAddonParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteAddonParams() *DeleteAddonParams {
	return &DeleteAddonParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteAddonParamsWithTimeout creates a new DeleteAddonParams object
// with the ability to set a timeout on a request.
func NewDeleteAddonParamsWithTimeout(timeout time.Duration) *DeleteAddonParams {
	return &DeleteAddonParams{
		timeout: timeout,
	}
}

// NewDeleteAddonParamsWithContext creates a new DeleteAddonParams object
// with the ability to set a context for a request.
func NewDeleteAddonParamsWithContext(ctx context.Context) *DeleteAddonParams {
	return &DeleteAddonParams{
		Context: ctx,
	}
}

// NewDeleteAddonParamsWithHTTPClient creates a new DeleteAddonParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteAddonParamsWithHTTPClient(client *http.Client) *DeleteAddonParams {
	return &DeleteAddonParams{
		HTTPClient: client,
	}
}

/* DeleteAddonParams contains all the parameters to send to the API endpoint
   for the delete addon operation.

   Typically these are written to a http.Request.
*/
type DeleteAddonParams struct {

	// AddonID.
	AddonID string

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

// WithDefaults hydrates default values in the delete addon params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteAddonParams) WithDefaults() *DeleteAddonParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete addon params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteAddonParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete addon params
func (o *DeleteAddonParams) WithTimeout(timeout time.Duration) *DeleteAddonParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete addon params
func (o *DeleteAddonParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete addon params
func (o *DeleteAddonParams) WithContext(ctx context.Context) *DeleteAddonParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete addon params
func (o *DeleteAddonParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete addon params
func (o *DeleteAddonParams) WithHTTPClient(client *http.Client) *DeleteAddonParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete addon params
func (o *DeleteAddonParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAddonID adds the addonID to the delete addon params
func (o *DeleteAddonParams) WithAddonID(addonID string) *DeleteAddonParams {
	o.SetAddonID(addonID)
	return o
}

// SetAddonID adds the addonId to the delete addon params
func (o *DeleteAddonParams) SetAddonID(addonID string) {
	o.AddonID = addonID
}

// WithClusterID adds the clusterID to the delete addon params
func (o *DeleteAddonParams) WithClusterID(clusterID string) *DeleteAddonParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the delete addon params
func (o *DeleteAddonParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the delete addon params
func (o *DeleteAddonParams) WithDC(dc string) *DeleteAddonParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the delete addon params
func (o *DeleteAddonParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the delete addon params
func (o *DeleteAddonParams) WithProjectID(projectID string) *DeleteAddonParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the delete addon params
func (o *DeleteAddonParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteAddonParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
