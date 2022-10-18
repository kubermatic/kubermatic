// Code generated by go-swagger; DO NOT EDIT.

package packet

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

// NewListPacketSizesParams creates a new ListPacketSizesParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewListPacketSizesParams() *ListPacketSizesParams {
	return &ListPacketSizesParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewListPacketSizesParamsWithTimeout creates a new ListPacketSizesParams object
// with the ability to set a timeout on a request.
func NewListPacketSizesParamsWithTimeout(timeout time.Duration) *ListPacketSizesParams {
	return &ListPacketSizesParams{
		timeout: timeout,
	}
}

// NewListPacketSizesParamsWithContext creates a new ListPacketSizesParams object
// with the ability to set a context for a request.
func NewListPacketSizesParamsWithContext(ctx context.Context) *ListPacketSizesParams {
	return &ListPacketSizesParams{
		Context: ctx,
	}
}

// NewListPacketSizesParamsWithHTTPClient creates a new ListPacketSizesParams object
// with the ability to set a custom HTTPClient for a request.
func NewListPacketSizesParamsWithHTTPClient(client *http.Client) *ListPacketSizesParams {
	return &ListPacketSizesParams{
		HTTPClient: client,
	}
}

/*
ListPacketSizesParams contains all the parameters to send to the API endpoint

	for the list packet sizes operation.

	Typically these are written to a http.Request.
*/
type ListPacketSizesParams struct {

	// APIKey.
	APIKey *string

	// Credential.
	Credential *string

	// ProjectID.
	ProjectID *string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the list packet sizes params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListPacketSizesParams) WithDefaults() *ListPacketSizesParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the list packet sizes params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *ListPacketSizesParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the list packet sizes params
func (o *ListPacketSizesParams) WithTimeout(timeout time.Duration) *ListPacketSizesParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the list packet sizes params
func (o *ListPacketSizesParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the list packet sizes params
func (o *ListPacketSizesParams) WithContext(ctx context.Context) *ListPacketSizesParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the list packet sizes params
func (o *ListPacketSizesParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the list packet sizes params
func (o *ListPacketSizesParams) WithHTTPClient(client *http.Client) *ListPacketSizesParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the list packet sizes params
func (o *ListPacketSizesParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithAPIKey adds the aPIKey to the list packet sizes params
func (o *ListPacketSizesParams) WithAPIKey(aPIKey *string) *ListPacketSizesParams {
	o.SetAPIKey(aPIKey)
	return o
}

// SetAPIKey adds the apiKey to the list packet sizes params
func (o *ListPacketSizesParams) SetAPIKey(aPIKey *string) {
	o.APIKey = aPIKey
}

// WithCredential adds the credential to the list packet sizes params
func (o *ListPacketSizesParams) WithCredential(credential *string) *ListPacketSizesParams {
	o.SetCredential(credential)
	return o
}

// SetCredential adds the credential to the list packet sizes params
func (o *ListPacketSizesParams) SetCredential(credential *string) {
	o.Credential = credential
}

// WithProjectID adds the projectID to the list packet sizes params
func (o *ListPacketSizesParams) WithProjectID(projectID *string) *ListPacketSizesParams {
	o.SetProjectID(projectID)
	return o
}

// SetProjectID adds the projectId to the list packet sizes params
func (o *ListPacketSizesParams) SetProjectID(projectID *string) {
	o.ProjectID = projectID
}

// WriteToRequest writes these params to a swagger request
func (o *ListPacketSizesParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.APIKey != nil {

		// header param apiKey
		if err := r.SetHeaderParam("apiKey", *o.APIKey); err != nil {
			return err
		}
	}

	if o.Credential != nil {

		// header param credential
		if err := r.SetHeaderParam("credential", *o.Credential); err != nil {
			return err
		}
	}

	if o.ProjectID != nil {

		// header param projectID
		if err := r.SetHeaderParam("projectID", *o.ProjectID); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
