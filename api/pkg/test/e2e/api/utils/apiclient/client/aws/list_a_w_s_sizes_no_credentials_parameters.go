// Code generated by go-swagger; DO NOT EDIT.

package aws

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

// NewListAWSSizesNoCredentialsParams creates a new ListAWSSizesNoCredentialsParams object
// with the default values initialized.
func NewListAWSSizesNoCredentialsParams() *ListAWSSizesNoCredentialsParams {
	var ()
	return &ListAWSSizesNoCredentialsParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewListAWSSizesNoCredentialsParamsWithTimeout creates a new ListAWSSizesNoCredentialsParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListAWSSizesNoCredentialsParamsWithTimeout(timeout time.Duration) *ListAWSSizesNoCredentialsParams {
	var ()
	return &ListAWSSizesNoCredentialsParams{

		timeout: timeout,
	}
}

// NewListAWSSizesNoCredentialsParamsWithContext creates a new ListAWSSizesNoCredentialsParams object
// with the default values initialized, and the ability to set a context for a request
func NewListAWSSizesNoCredentialsParamsWithContext(ctx context.Context) *ListAWSSizesNoCredentialsParams {
	var ()
	return &ListAWSSizesNoCredentialsParams{

		Context: ctx,
	}
}

// NewListAWSSizesNoCredentialsParamsWithHTTPClient creates a new ListAWSSizesNoCredentialsParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListAWSSizesNoCredentialsParamsWithHTTPClient(client *http.Client) *ListAWSSizesNoCredentialsParams {
	var ()
	return &ListAWSSizesNoCredentialsParams{
		HTTPClient: client,
	}
}

/*ListAWSSizesNoCredentialsParams contains all the parameters to send to the API endpoint
for the list a w s sizes no credentials operation typically these are written to a http.Request
*/
type ListAWSSizesNoCredentialsParams struct {

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

// WithTimeout adds the timeout to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) WithTimeout(timeout time.Duration) *ListAWSSizesNoCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) WithContext(ctx context.Context) *ListAWSSizesNoCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) WithHTTPClient(client *http.Client) *ListAWSSizesNoCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) WithClusterID(clusterID string) *ListAWSSizesNoCredentialsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) WithDC(dc string) *ListAWSSizesNoCredentialsParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) WithProjectID(projectID string) *ListAWSSizesNoCredentialsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list a w s sizes no credentials params
func (o *ListAWSSizesNoCredentialsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListAWSSizesNoCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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
