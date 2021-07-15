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

// NewGetEtcdBackupConfigParams creates a new GetEtcdBackupConfigParams object
// with the default values initialized.
func NewGetEtcdBackupConfigParams() *GetEtcdBackupConfigParams {
	var ()
	return &GetEtcdBackupConfigParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetEtcdBackupConfigParamsWithTimeout creates a new GetEtcdBackupConfigParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetEtcdBackupConfigParamsWithTimeout(timeout time.Duration) *GetEtcdBackupConfigParams {
	var ()
	return &GetEtcdBackupConfigParams{

		timeout: timeout,
	}
}

// NewGetEtcdBackupConfigParamsWithContext creates a new GetEtcdBackupConfigParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetEtcdBackupConfigParamsWithContext(ctx context.Context) *GetEtcdBackupConfigParams {
	var ()
	return &GetEtcdBackupConfigParams{

		Context: ctx,
	}
}

// NewGetEtcdBackupConfigParamsWithHTTPClient creates a new GetEtcdBackupConfigParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetEtcdBackupConfigParamsWithHTTPClient(client *http.Client) *GetEtcdBackupConfigParams {
	var ()
	return &GetEtcdBackupConfigParams{
		HTTPClient: client,
	}
}

/*GetEtcdBackupConfigParams contains all the parameters to send to the API endpoint
for the get etcd backup config operation typically these are written to a http.Request
*/
type GetEtcdBackupConfigParams struct {

	/*ClusterID*/
	ClusterID string
	/*EbcName*/
	EtcdBackupConfigName string
	/*ProjectID*/
	ProjectID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) WithTimeout(timeout time.Duration) *GetEtcdBackupConfigParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) WithContext(ctx context.Context) *GetEtcdBackupConfigParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) WithHTTPClient(client *http.Client) *GetEtcdBackupConfigParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithClusterID adds the clusterID to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) WithClusterID(clusterID string) *GetEtcdBackupConfigParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithEtcdBackupConfigName adds the ebcName to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) WithEtcdBackupConfigName(ebcName string) *GetEtcdBackupConfigParams {
	o.SetEtcdBackupConfigName(ebcName)
	return o
}

// SetEtcdBackupConfigName adds the ebcName to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) SetEtcdBackupConfigName(ebcName string) {
	o.EtcdBackupConfigName = ebcName
}

// WithProjectID adds the projectID to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) WithProjectID(projectID string) *GetEtcdBackupConfigParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the get etcd backup config params
func (o *GetEtcdBackupConfigParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *GetEtcdBackupConfigParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param cluster_id
	if err := r.SetPathParam("cluster_id", o.ClusterID); err != nil {
		return err
	}

	// path param ebc_name
	if err := r.SetPathParam("ebc_name", o.EtcdBackupConfigName); err != nil {
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
