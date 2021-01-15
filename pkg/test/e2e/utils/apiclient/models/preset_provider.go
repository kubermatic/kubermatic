// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// PresetProvider PresetProvider represents a preset provider
//
// swagger:model PresetProvider
type PresetProvider struct {

	// enabled
	Enabled bool `json:"enabled,omitempty"`

	// name
	Name ProviderType `json:"name,omitempty"`
}

// Validate validates this preset provider
func (m *PresetProvider) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateName(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *PresetProvider) validateName(formats strfmt.Registry) error {

	if swag.IsZero(m.Name) { // not required
		return nil
	}

	if err := m.Name.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("name")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *PresetProvider) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *PresetProvider) UnmarshalBinary(b []byte) error {
	var res PresetProvider
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
