// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// OPAIntegrationSettings o p a integration settings
//
// swagger:model OPAIntegrationSettings
type OPAIntegrationSettings struct {

	// Enabled is the flag for enabling OPA integration
	Enabled bool `json:"enabled,omitempty"`

	// WebhookTimeout is the timeout that is set for the gatekeeper validating webhook admission review calls.
	// By default 10 seconds.
	WebhookTimeoutSeconds int32 `json:"webhookTimeoutSeconds,omitempty"`
}

// Validate validates this o p a integration settings
func (m *OPAIntegrationSettings) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this o p a integration settings based on context it is used
func (m *OPAIntegrationSettings) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *OPAIntegrationSettings) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *OPAIntegrationSettings) UnmarshalBinary(b []byte) error {
	var res OPAIntegrationSettings
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
