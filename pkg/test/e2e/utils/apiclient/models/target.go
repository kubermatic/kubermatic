// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// Target target
//
// swagger:model Target
type Target struct {

	// libs
	Libs []string `json:"libs"`

	// rego
	Rego string `json:"rego,omitempty"`

	// target
	Target string `json:"target,omitempty"`
}

// Validate validates this target
func (m *Target) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this target based on context it is used
func (m *Target) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *Target) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Target) UnmarshalBinary(b []byte) error {
	var res Target
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
