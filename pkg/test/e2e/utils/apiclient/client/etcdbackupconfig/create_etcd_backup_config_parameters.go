// Code generated by go-swagger; DO NOT EDIT.

package etcdbackupconfig

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

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// NewCreateEtcdBackupConfigParams creates a new CreateEtcdBackupConfigParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewCreateEtcdBackupConfigParams() *CreateEtcdBackupConfigParams {
	return &CreateEtcdBackupConfigParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewCreateEtcdBackupConfigParamsWithTimeout creates a new CreateEtcdBackupConfigParams object
// with the ability to set a timeout on a request.
func NewCreateEtcdBackupConfigParamsWithTimeout(timeout time.Duration) *CreateEtcdBackupConfigParams {
	return &CreateEtcdBackupConfigParams{
		timeout: timeout,
	}
}

// NewCreateEtcdBackupConfigParamsWithContext creates a new CreateEtcdBackupConfigParams object
// with the ability to set a context for a request.
func NewCreateEtcdBackupConfigParamsWithContext(ctx context.Context) *CreateEtcdBackupConfigParams {
	return &CreateEtcdBackupConfigParams{
		Context: ctx,
	}
}

// NewCreateEtcdBackupConfigParamsWithHTTPClient creates a new CreateEtcdBackupConfigParams object
// with the ability to set a custom HTTPClient for a request.
func NewCreateEtcdBackupConfigParamsWithHTTPClient(client *http.Client) *CreateEtcdBackupConfigParams {
	return &CreateEtcdBackupConfigParams{
		HTTPClient: client,
	}
}

/*
CreateEtcdBackupConfigParams contains all the parameters to send to the API endpoint

	for the create etcd backup config operation.

	Typically these are written to a http.Request.
*/
type CreateEtcdBackupConfigParams struct {

	// Body.
	Body *models.EbcBody

	// ClusterID.
	ClusterID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the create etcd backup config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateEtcdBackupConfigParams) WithDefaults() *CreateEtcdBackupConfigParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the create etcd backup config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *CreateEtcdBackupConfigParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) WithTimeout(timeout time.Duration) *CreateEtcdBackupConfigParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) WithContext(ctx context.Context) *CreateEtcdBackupConfigParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) WithHTTPClient(client *http.Client) *CreateEtcdBackupConfigParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) WithBody(body *models.EbcBody) *CreateEtcdBackupConfigParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) SetBody(body *models.EbcBody) {
	o.Body = body
}

// WithClusterID adds the clusterID to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) WithClusterID(clusterID string) *CreateEtcdBackupConfigParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithProjectID adds the projectID to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) WithProjectID(projectID string) *CreateEtcdBackupConfigParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the create etcd backup config params
func (o *CreateEtcdBackupConfigParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *CreateEtcdBackupConfigParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

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

	// path param project_id
	if err := r.SetPathParam("project_id", o.ProjectID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
