// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/swag"
)

// AuthProviderConfig AuthProviderConfig holds the configuration for a specified auth provider.
// swagger:model AuthProviderConfig
type AuthProviderConfig struct {

	// config
	Config map[string]string `json:"config,omitempty"`

	// name
	Name string `json:"name,omitempty"`
}

// Validate validates this auth provider config
func (m *AuthProviderConfig) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AuthProviderConfig) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AuthProviderConfig) UnmarshalBinary(b []byte) error {
	var res AuthProviderConfig
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
