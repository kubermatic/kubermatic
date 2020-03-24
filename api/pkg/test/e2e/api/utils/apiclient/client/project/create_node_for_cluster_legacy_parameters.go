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

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// NewCreateNodeForClusterLegacyParams creates a new CreateNodeForClusterLegacyParams object
// with the default values initialized.
func NewCreateNodeForClusterLegacyParams() *CreateNodeForClusterLegacyParams {
	var ()
	return &CreateNodeForClusterLegacyParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewCreateNodeForClusterLegacyParamsWithTimeout creates a new CreateNodeForClusterLegacyParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewCreateNodeForClusterLegacyParamsWithTimeout(timeout time.Duration) *CreateNodeForClusterLegacyParams {
	var ()
	return &CreateNodeForClusterLegacyParams{

		timeout: timeout,
	}
}

// NewCreateNodeForClusterLegacyParamsWithContext creates a new CreateNodeForClusterLegacyParams object
// with the default values initialized, and the ability to set a context for a request
func NewCreateNodeForClusterLegacyParamsWithContext(ctx context.Context) *CreateNodeForClusterLegacyParams {
	var ()
	return &CreateNodeForClusterLegacyParams{

		Context: ctx,
	}
}

// NewCreateNodeForClusterLegacyParamsWithHTTPClient creates a new CreateNodeForClusterLegacyParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewCreateNodeForClusterLegacyParamsWithHTTPClient(client *http.Client) *CreateNodeForClusterLegacyParams {
	var ()
	return &CreateNodeForClusterLegacyParams{
		HTTPClient: client,
	}
}

/*CreateNodeForClusterLegacyParams contains all the parameters to send to the API endpoint
for the create node for cluster legacy operation typically these are written to a http.Request
*/
type CreateNodeForClusterLegacyParams struct {

	/*Body*/
	Body *models.Node
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

// WithTimeout adds the timeout to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) WithTimeout(timeout time.Duration) *CreateNodeForClusterLegacyParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) WithContext(ctx context.Context) *CreateNodeForClusterLegacyParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) WithHTTPClient(client *http.Client) *CreateNodeForClusterLegacyParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) WithBody(body *models.Node) *CreateNodeForClusterLegacyParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) SetBody(body *models.Node) {
	o.Body = body
}

// WithClusterID adds the clusterID to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) WithClusterID(clusterID string) *CreateNodeForClusterLegacyParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) WithDC(dc string) *CreateNodeForClusterLegacyParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) WithProjectID(projectID string) *CreateNodeForClusterLegacyParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the create node for cluster legacy params
func (o *CreateNodeForClusterLegacyParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *CreateNodeForClusterLegacyParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
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
