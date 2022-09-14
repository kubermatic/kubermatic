// Code generated by go-swagger; DO NOT EDIT.

package rulegroup

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

// NewUpdateAdminRuleGroupParams creates a new UpdateAdminRuleGroupParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewUpdateAdminRuleGroupParams() *UpdateAdminRuleGroupParams {
	return &UpdateAdminRuleGroupParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewUpdateAdminRuleGroupParamsWithTimeout creates a new UpdateAdminRuleGroupParams object
// with the ability to set a timeout on a request.
func NewUpdateAdminRuleGroupParamsWithTimeout(timeout time.Duration) *UpdateAdminRuleGroupParams {
	return &UpdateAdminRuleGroupParams{
		timeout: timeout,
	}
}

// NewUpdateAdminRuleGroupParamsWithContext creates a new UpdateAdminRuleGroupParams object
// with the ability to set a context for a request.
func NewUpdateAdminRuleGroupParamsWithContext(ctx context.Context) *UpdateAdminRuleGroupParams {
	return &UpdateAdminRuleGroupParams{
		Context: ctx,
	}
}

// NewUpdateAdminRuleGroupParamsWithHTTPClient creates a new UpdateAdminRuleGroupParams object
// with the ability to set a custom HTTPClient for a request.
func NewUpdateAdminRuleGroupParamsWithHTTPClient(client *http.Client) *UpdateAdminRuleGroupParams {
	return &UpdateAdminRuleGroupParams{
		HTTPClient: client,
	}
}

/*
UpdateAdminRuleGroupParams contains all the parameters to send to the API endpoint

	for the update admin rule group operation.

	Typically these are written to a http.Request.
*/
type UpdateAdminRuleGroupParams struct {

	// Body.
	Body *models.RuleGroup

	// RulegroupID.
	RuleGroupID string

	// SeedName.
	SeedName string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the update admin rule group params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpdateAdminRuleGroupParams) WithDefaults() *UpdateAdminRuleGroupParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the update admin rule group params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *UpdateAdminRuleGroupParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) WithTimeout(timeout time.Duration) *UpdateAdminRuleGroupParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) WithContext(ctx context.Context) *UpdateAdminRuleGroupParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) WithHTTPClient(client *http.Client) *UpdateAdminRuleGroupParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) WithBody(body *models.RuleGroup) *UpdateAdminRuleGroupParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) SetBody(body *models.RuleGroup) {
	o.Body = body
}

// WithRuleGroupID adds the rulegroupID to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) WithRuleGroupID(rulegroupID string) *UpdateAdminRuleGroupParams {
	o.SetRuleGroupID(rulegroupID)
	return o
}

// SetRuleGroupID adds the rulegroupId to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) SetRuleGroupID(rulegroupID string) {
	o.RuleGroupID = rulegroupID
}

// WithSeedName adds the seedName to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) WithSeedName(seedName string) *UpdateAdminRuleGroupParams {
	o.SetSeedName(seedName)
	return o
}

// SetSeedName adds the seedName to the update admin rule group params
func (o *UpdateAdminRuleGroupParams) SetSeedName(seedName string) {
	o.SeedName = seedName
}

// WriteToRequest writes these params to a swagger request
func (o *UpdateAdminRuleGroupParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error
	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	// path param rulegroup_id
	if err := r.SetPathParam("rulegroup_id", o.RuleGroupID); err != nil {
		return err
	}

	// path param seed_name
	if err := r.SetPathParam("seed_name", o.SeedName); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
