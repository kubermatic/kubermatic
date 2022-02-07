// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// SeedMLASettings SeedMLASettings allow configuring seed level MLA (Monitoring, Logging & Alerting) stack settings.
//
// swagger:model SeedMLASettings
type SeedMLASettings struct {

	// Optional: UserClusterMLAEnabled controls whether the user cluster MLA (Monitoring, Logging & Alerting) stack is enabled in the seed.
	UserClusterMLAEnabled bool `json:"userClusterMLAEnabled,omitempty"`
}

// Validate validates this seed m l a settings
func (m *SeedMLASettings) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this seed m l a settings based on context it is used
func (m *SeedMLASettings) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *SeedMLASettings) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *SeedMLASettings) UnmarshalBinary(b []byte) error {
	var res SeedMLASettings
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
