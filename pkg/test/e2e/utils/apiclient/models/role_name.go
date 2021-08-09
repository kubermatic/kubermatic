// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// RoleName RoleName defines RBAC role name object for the user cluster
//
// swagger:model RoleName
type RoleName struct {

	// Name of the role.
	Name string `json:"name,omitempty"`

	// Indicates the scopes of this role.
	Namespace []string `json:"namespace"`
}

// Validate validates this role name
func (m *RoleName) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this role name based on context it is used
func (m *RoleName) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *RoleName) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *RoleName) UnmarshalBinary(b []byte) error {
	var res RoleName
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
