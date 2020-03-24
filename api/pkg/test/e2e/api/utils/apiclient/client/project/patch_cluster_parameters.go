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

// NewPatchClusterParams creates a new PatchClusterParams object
// with the default values initialized.
func NewPatchClusterParams() *PatchClusterParams {
	var ()
	return &PatchClusterParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewPatchClusterParamsWithTimeout creates a new PatchClusterParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewPatchClusterParamsWithTimeout(timeout time.Duration) *PatchClusterParams {
	var ()
	return &PatchClusterParams{

		timeout: timeout,
	}
}

// NewPatchClusterParamsWithContext creates a new PatchClusterParams object
// with the default values initialized, and the ability to set a context for a request
func NewPatchClusterParamsWithContext(ctx context.Context) *PatchClusterParams {
	var ()
	return &PatchClusterParams{

		Context: ctx,
	}
}

// NewPatchClusterParamsWithHTTPClient creates a new PatchClusterParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewPatchClusterParamsWithHTTPClient(client *http.Client) *PatchClusterParams {
	var ()
	return &PatchClusterParams{
		HTTPClient: client,
	}
}

/*PatchClusterParams contains all the parameters to send to the API endpoint
for the patch cluster operation typically these are written to a http.Request
*/
type PatchClusterParams struct {

	/*Patch*/
	Patch interface{}
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

// WithTimeout adds the timeout to the patch cluster params
func (o *PatchClusterParams) WithTimeout(timeout time.Duration) *PatchClusterParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the patch cluster params
func (o *PatchClusterParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the patch cluster params
func (o *PatchClusterParams) WithContext(ctx context.Context) *PatchClusterParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the patch cluster params
func (o *PatchClusterParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the patch cluster params
func (o *PatchClusterParams) WithHTTPClient(client *http.Client) *PatchClusterParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the patch cluster params
func (o *PatchClusterParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithPatch adds the patch to the patch cluster params
func (o *PatchClusterParams) WithPatch(patch interface{}) *PatchClusterParams {
	o.SetPatch(patch)
	return o
}

// SetPatch adds the patch to the patch cluster params
func (o *PatchClusterParams) SetPatch(patch interface{}) {
	o.Patch = patch
}

// WithClusterID adds the clusterID to the patch cluster params
func (o *PatchClusterParams) WithClusterID(clusterID string) *PatchClusterParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the patch cluster params
func (o *PatchClusterParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the patch cluster params
func (o *PatchClusterParams) WithDC(dc string) *PatchClusterParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the patch cluster params
func (o *PatchClusterParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the patch cluster params
func (o *PatchClusterParams) WithProjectID(projectID string) *PatchClusterParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the patch cluster params
func (o *PatchClusterParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *PatchClusterParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Patch != nil {
		if err := r.SetBodyParam(o.Patch); err != nil {
			return err
		}
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
