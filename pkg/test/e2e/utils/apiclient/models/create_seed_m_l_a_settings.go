// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// CreateSeedMLASettings create seed m l a settings
//
// swagger:model CreateSeedMLASettings
type CreateSeedMLASettings struct {

	// Optional: UserClusterMLAEnabled controls whether the user cluster MLA (Monitoring, Logging & Alerting) stack is enabled in the seed.
	UserClusterMLAEnabled bool `json:"userClusterMLAEnabled,omitempty"`
}

// Validate validates this create seed m l a settings
func (m *CreateSeedMLASettings) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this create seed m l a settings based on context it is used
func (m *CreateSeedMLASettings) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *CreateSeedMLASettings) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *CreateSeedMLASettings) UnmarshalBinary(b []byte) error {
	var res CreateSeedMLASettings
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
