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
)

// NewDeleteEtcdBackupConfigParams creates a new DeleteEtcdBackupConfigParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteEtcdBackupConfigParams() *DeleteEtcdBackupConfigParams {
	return &DeleteEtcdBackupConfigParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteEtcdBackupConfigParamsWithTimeout creates a new DeleteEtcdBackupConfigParams object
// with the ability to set a timeout on a request.
func NewDeleteEtcdBackupConfigParamsWithTimeout(timeout time.Duration) *DeleteEtcdBackupConfigParams {
	return &DeleteEtcdBackupConfigParams{
		timeout: timeout,
	}
}

// NewDeleteEtcdBackupConfigParamsWithContext creates a new DeleteEtcdBackupConfigParams object
// with the ability to set a context for a request.
func NewDeleteEtcdBackupConfigParamsWithContext(ctx context.Context) *DeleteEtcdBackupConfigParams {
	return &DeleteEtcdBackupConfigParams{
		Context: ctx,
	}
}

// NewDeleteEtcdBackupConfigParamsWithHTTPClient creates a new DeleteEtcdBackupConfigParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteEtcdBackupConfigParamsWithHTTPClient(client *http.Client) *DeleteEtcdBackupConfigParams {
	return &DeleteEtcdBackupConfigParams{
		HTTPClient: client,
	}
}

/*
DeleteEtcdBackupConfigParams contains all the parameters to send to the API endpoint

	for the delete etcd backup config operation.

	Typically these are written to a http.Request.
*/
type DeleteEtcdBackupConfigParams struct {

	// ClusterID.
	ClusterID string

	// EbcID.
	EtcdBackupConfigID string

	// ProjectID.
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete etcd backup config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteEtcdBackupConfigParams) WithDefaults() *DeleteEtcdBackupConfigParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete etcd backup config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteEtcdBackupConfigParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) WithTimeout(timeout time.Duration) *DeleteEtcdBackupConfigParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) WithContext(ctx context.Context) *DeleteEtcdBackupConfigParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) WithHTTPClient(client *http.Client) *DeleteEtcdBackupConfigParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) WithClusterID(clusterID string) *DeleteEtcdBackupConfigParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithEtcdBackupConfigID adds the ebcID to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) WithEtcdBackupConfigID(ebcID string) *DeleteEtcdBackupConfigParams {
	o.SetEtcdBackupConfigID(ebcID)
	return o
}

// SetEtcdBackupConfigID adds the ebcId to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) SetEtcdBackupConfigID(ebcID string) {
	o.EtcdBackupConfigID = ebcID
}

// WithProjectID adds the projectID to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) WithProjectID(projectID string) *DeleteEtcdBackupConfigParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the delete etcd backup config params
func (o *DeleteEtcdBackupConfigParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteEtcdBackupConfigParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
		return err
	}

	// path param ebc_id
	if err := r.SetPathParam("ebc_id", o.EtcdBackupConfigID); err != nil {
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
