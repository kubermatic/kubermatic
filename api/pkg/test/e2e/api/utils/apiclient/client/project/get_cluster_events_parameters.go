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

// NewGetClusterEventsParams creates a new GetClusterEventsParams object
// with the default values initialized.
func NewGetClusterEventsParams() *GetClusterEventsParams {
	var ()
	return &GetClusterEventsParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetClusterEventsParamsWithTimeout creates a new GetClusterEventsParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetClusterEventsParamsWithTimeout(timeout time.Duration) *GetClusterEventsParams {
	var ()
	return &GetClusterEventsParams{

		timeout: timeout,
	}
}

// NewGetClusterEventsParamsWithContext creates a new GetClusterEventsParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetClusterEventsParamsWithContext(ctx context.Context) *GetClusterEventsParams {
	var ()
	return &GetClusterEventsParams{

		Context: ctx,
	}
}

// NewGetClusterEventsParamsWithHTTPClient creates a new GetClusterEventsParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetClusterEventsParamsWithHTTPClient(client *http.Client) *GetClusterEventsParams {
	var ()
	return &GetClusterEventsParams{
		HTTPClient: client,
	}
}

/*GetClusterEventsParams contains all the parameters to send to the API endpoint
for the get cluster events operation typically these are written to a http.Request
*/
type GetClusterEventsParams struct {

	/*ClusterID*/
	ClusterID string
	/*Dc*/
	DC string
	/*ProjectID*/
	ProjectID string
	/*Type*/
	Type *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get cluster events params
func (o *GetClusterEventsParams) WithTimeout(timeout time.Duration) *GetClusterEventsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get cluster events params
func (o *GetClusterEventsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get cluster events params
func (o *GetClusterEventsParams) WithContext(ctx context.Context) *GetClusterEventsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get cluster events params
func (o *GetClusterEventsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get cluster events params
func (o *GetClusterEventsParams) WithHTTPClient(client *http.Client) *GetClusterEventsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get cluster events params
func (o *GetClusterEventsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the get cluster events params
func (o *GetClusterEventsParams) WithClusterID(clusterID string) *GetClusterEventsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get cluster events params
func (o *GetClusterEventsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the get cluster events params
func (o *GetClusterEventsParams) WithDC(dc string) *GetClusterEventsParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the get cluster events params
func (o *GetClusterEventsParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the get cluster events params
func (o *GetClusterEventsParams) WithProjectID(projectID string) *GetClusterEventsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get cluster events params
func (o *GetClusterEventsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithType adds the typeVar to the get cluster events params
func (o *GetClusterEventsParams) WithType(typeVar *string) *GetClusterEventsParams {
	o.SetType(typeVar)
	return o
}

// SetType adds the type to the get cluster events params
func (o *GetClusterEventsParams) SetType(typeVar *string) {
	o.Type = typeVar
}

// WriteToRequest writes these params to a swagger request
func (o *GetClusterEventsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	if o.Type != nil {

		// query param type
		var qrType string
		if o.Type != nil {
			qrType = *o.Type
		}
		qType := qrType
		if qType != "" {
			if err := r.SetQueryParam("type", qType); err != nil {
				return err
			}
		}

	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
