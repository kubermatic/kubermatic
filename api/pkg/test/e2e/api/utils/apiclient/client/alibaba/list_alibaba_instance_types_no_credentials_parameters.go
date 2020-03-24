// Code generated by go-swagger; DO NOT EDIT.

package alibaba

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

// NewListAlibabaInstanceTypesNoCredentialsParams creates a new ListAlibabaInstanceTypesNoCredentialsParams object
// with the default values initialized.
func NewListAlibabaInstanceTypesNoCredentialsParams() *ListAlibabaInstanceTypesNoCredentialsParams {
	var ()
	return &ListAlibabaInstanceTypesNoCredentialsParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewListAlibabaInstanceTypesNoCredentialsParamsWithTimeout creates a new ListAlibabaInstanceTypesNoCredentialsParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewListAlibabaInstanceTypesNoCredentialsParamsWithTimeout(timeout time.Duration) *ListAlibabaInstanceTypesNoCredentialsParams {
	var ()
	return &ListAlibabaInstanceTypesNoCredentialsParams{

		timeout: timeout,
	}
}

// NewListAlibabaInstanceTypesNoCredentialsParamsWithContext creates a new ListAlibabaInstanceTypesNoCredentialsParams object
// with the default values initialized, and the ability to set a context for a request
func NewListAlibabaInstanceTypesNoCredentialsParamsWithContext(ctx context.Context) *ListAlibabaInstanceTypesNoCredentialsParams {
	var ()
	return &ListAlibabaInstanceTypesNoCredentialsParams{

		Context: ctx,
	}
}

// NewListAlibabaInstanceTypesNoCredentialsParamsWithHTTPClient creates a new ListAlibabaInstanceTypesNoCredentialsParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewListAlibabaInstanceTypesNoCredentialsParamsWithHTTPClient(client *http.Client) *ListAlibabaInstanceTypesNoCredentialsParams {
	var ()
	return &ListAlibabaInstanceTypesNoCredentialsParams{
		HTTPClient: client,
	}
}

/*ListAlibabaInstanceTypesNoCredentialsParams contains all the parameters to send to the API endpoint
for the list alibaba instance types no credentials operation typically these are written to a http.Request
*/
type ListAlibabaInstanceTypesNoCredentialsParams struct {

	/*Region*/
	Region *string
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

// WithTimeout adds the timeout to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) WithTimeout(timeout time.Duration) *ListAlibabaInstanceTypesNoCredentialsParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) WithContext(ctx context.Context) *ListAlibabaInstanceTypesNoCredentialsParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) WithHTTPClient(client *http.Client) *ListAlibabaInstanceTypesNoCredentialsParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithRegion adds the region to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) WithRegion(region *string) *ListAlibabaInstanceTypesNoCredentialsParams {
	o.SetRegion(region)
	return o
}

// SetRegion adds the region to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) SetRegion(region *string) {
	o.Region = region
}

// WithClusterID adds the clusterID to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) WithClusterID(clusterID string) *ListAlibabaInstanceTypesNoCredentialsParams {
	o.SetClusterID(clusterID)
	return o
}

// SetClusterID adds the clusterId to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) SetClusterID(clusterID string) {
	o.ClusterID = clusterID
}

// WithDC adds the dc to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) WithDC(dc string) *ListAlibabaInstanceTypesNoCredentialsParams {
	o.SetDC(dc)
	return o
}

// SetDC adds the dc to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) SetDC(dc string) {
	o.DC = dc
}

// WithProjectID adds the projectID to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) WithProjectID(projectID string) *ListAlibabaInstanceTypesNoCredentialsParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list alibaba instance types no credentials params
func (o *ListAlibabaInstanceTypesNoCredentialsParams) SetProjectID(projectID string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListAlibabaInstanceTypesNoCredentialsParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Region != nil {

		// header param Region
		if err := r.SetHeaderParam("Region", *o.Region); err != nil {
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
