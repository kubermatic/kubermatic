// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// SecretReference SecretReference represents a Secret Reference. It has enough information to retrieve secret
// in any namespace
// +structType=atomic
//
// swagger:model SecretReference
type SecretReference struct {

	// name is unique within a namespace to reference a secret resource.
	// +optional
	Name string `json:"name,omitempty"`

	// namespace defines the space within which the secret name must be unique.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// Validate validates this secret reference
func (m *SecretReference) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this secret reference based on context it is used
func (m *SecretReference) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *SecretReference) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *SecretReference) UnmarshalBinary(b []byte) error {
	var res SecretReference
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
