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

// NewListProjectEtcdBackupConfigParams creates a new ListProjectEtcdBackupConfigParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListProjectEtcdBackupConfigParams() *ListProjectEtcdBackupConfigParams {
	return &ListProjectEtcdBackupConfigParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListProjectEtcdBackupConfigParamsWithTimeout creates a new ListProjectEtcdBackupConfigParams object
// with the ability to set a timeout on a request.
func NewListProjectEtcdBackupConfigParamsWithTimeout(timeout time.Duration) *ListProjectEtcdBackupConfigParams {
	return &ListProjectEtcdBackupConfigParams{
		timeout: timeout,
	}
}

// NewListProjectEtcdBackupConfigParamsWithContext creates a new ListProjectEtcdBackupConfigParams object
// with the ability to set a context for a request.
func NewListProjectEtcdBackupConfigParamsWithContext(ctx context.Context) *ListProjectEtcdBackupConfigParams {
	return &ListProjectEtcdBackupConfigParams{
		Context: ctx,
	}
}

// NewListProjectEtcdBackupConfigParamsWithHTTPClient creates a new ListProjectEtcdBackupConfigParams object
// with the ability to set a custom HTTPClient for a request.
func NewListProjectEtcdBackupConfigParamsWithHTTPClient(client *http.Client) *ListProjectEtcdBackupConfigParams {
	return &ListProjectEtcdBackupConfigParams{
		HTTPClient: client,
	}
}

/*
ListProjectEtcdBackupConfigParams contains all the parameters to send to the API endpoint

	for the list project etcd backup config operation.

	Typically these are written to a http.Request.
*/
type ListProjectEtcdBackupConfigParams struct {

	// ProjectID.
	ProjectID string

	// Type.
	Type *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list project etcd backup config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListProjectEtcdBackupConfigParams) WithDefaults() *ListProjectEtcdBackupConfigParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list project etcd backup config params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListProjectEtcdBackupConfigParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list project etcd backup config params
func (o *ListProjectEtcdBackupConfigParams) WithTimeout(timeout time.Duration) *ListProjectEtcdBackupConfigParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list project etcd backup config params
func (o *ListProjectEtcdBackupConfigParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list project etcd backup config params
func (o *ListProjectEtcdBackupConfigParams) WithContext(ctx context.Context) *ListProjectEtcdBackupConfigParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list project etcd backup config params
func (o *ListProjectEtcdBackupConfigParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list project etcd backup config params
func (o *ListProjectEtcdBackupConfigParams) WithHTTPClient(client *http.Client) *ListProjectEtcdBackupConfigParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list project etcd backup config params
func (o *ListProjectEtcdBackupConfigParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithProjectID adds the projectID to the list project etcd backup config params
func (o *ListProjectEtcdBackupConfigParams) WithProjectID(projectID string) *ListProjectEtcdBackupConfigParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list project etcd backup config params
func (o *ListProjectEtcdBackupConfigParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WithType adds the typeVar to the list project etcd backup config params
func (o *ListProjectEtcdBackupConfigParams) WithType(typeVar *string) *ListProjectEtcdBackupConfigParams {
	o.SetType(typeVar)
	return o
}

// SetType adds the type to the list project etcd backup config params
func (o *ListProjectEtcdBackupConfigParams) SetType(typeVar *string) {
	o.Type = typeVar
}

// WriteToRequest writes these params to a swagger request
func (o *ListProjectEtcdBackupConfigParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

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
