// Code generated by go-swagger; DO NOT EDIT.

package digitalocean

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

// NewListDigitaloceanSizesNoCredentialsV2Params creates a new ListDigitaloceanSizesNoCredentialsV2Params object
// with the default values initialized.
func NewListDigitaloceanSizesNoCredentialsV2Params() *ListDigitaloceanSizesNoCredentialsV2Params {
	var ()
	return &ListDigitaloceanSizesNoCredentialsV2Params{

		timeout: cr.DefaultTimeout,
	}
}

// NewListDigitaloceanSizesNoCredentialsV2ParamsWithTimeout creates a new ListDigitaloceanSizesNoCredentialsV2Params object
// with the default values initialized, and the ability to set a timeout on a request
func NewListDigitaloceanSizesNoCredentialsV2ParamsWithTimeout(timeout time.Duration) *ListDigitaloceanSizesNoCredentialsV2Params {
	var ()
	return &ListDigitaloceanSizesNoCredentialsV2Params{

		timeout: timeout,
	}
}

// NewListDigitaloceanSizesNoCredentialsV2ParamsWithContext creates a new ListDigitaloceanSizesNoCredentialsV2Params object
// with the default values initialized, and the ability to set a context for a request
func NewListDigitaloceanSizesNoCredentialsV2ParamsWithContext(ctx context.Context) *ListDigitaloceanSizesNoCredentialsV2Params {
	var ()
	return &ListDigitaloceanSizesNoCredentialsV2Params{

		Context: ctx,
	}
}

// NewListDigitaloceanSizesNoCredentialsV2ParamsWithHTTPClient creates a new ListDigitaloceanSizesNoCredentialsV2Params object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListDigitaloceanSizesNoCredentialsV2ParamsWithHTTPClient(client *http.Client) *ListDigitaloceanSizesNoCredentialsV2Params {
	var ()
	return &ListDigitaloceanSizesNoCredentialsV2Params{
		HTTPClient: client,
	}
}

/*ListDigitaloceanSizesNoCredentialsV2Params contains all the parameters to send to the API endpoint
for the list digitalocean sizes no credentials v2 operation typically these are written to a http.Request
*/
type ListDigitaloceanSizesNoCredentialsV2Params struct {

	/*ClusterID*/
	ClusterID string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the list digitalocean sizes no credentials v2 params
func (o *ListDigitaloceanSizesNoCredentialsV2Params) WithTimeout(timeout time.Duration) *ListDigitaloceanSizesNoCredentialsV2Params {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list digitalocean sizes no credentials v2 params
func (o *ListDigitaloceanSizesNoCredentialsV2Params) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list digitalocean sizes no credentials v2 params
func (o *ListDigitaloceanSizesNoCredentialsV2Params) WithContext(ctx context.Context) *ListDigitaloceanSizesNoCredentialsV2Params {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list digitalocean sizes no credentials v2 params
func (o *ListDigitaloceanSizesNoCredentialsV2Params) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list digitalocean sizes no credentials v2 params
func (o *ListDigitaloceanSizesNoCredentialsV2Params) WithHTTPClient(client *http.Client) *ListDigitaloceanSizesNoCredentialsV2Params {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list digitalocean sizes no credentials v2 params
func (o *ListDigitaloceanSizesNoCredentialsV2Params) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the list digitalocean sizes no credentials v2 params
func (o *ListDigitaloceanSizesNoCredentialsV2Params) WithClusterID(clusterID string) *ListDigitaloceanSizesNoCredentialsV2Params {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list digitalocean sizes no credentials v2 params
func (o *ListDigitaloceanSizesNoCredentialsV2Params) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the list digitalocean sizes no credentials v2 params
func (o *ListDigitaloceanSizesNoCredentialsV2Params) WithProjectID(projectID string) *ListDigitaloceanSizesNoCredentialsV2Params {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list digitalocean sizes no credentials v2 params
func (o *ListDigitaloceanSizesNoCredentialsV2Params) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListDigitaloceanSizesNoCredentialsV2Params) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

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
