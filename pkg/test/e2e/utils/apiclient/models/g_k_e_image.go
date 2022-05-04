// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// GKEImage GKEImage represents an object of GKE image.
//
// swagger:model GKEImage
type GKEImage struct {

	// is default
	IsDefault bool `json:"default,omitempty"`

	// name
	Name string `json:"name,omitempty"`
}

// Validate validates this g k e image
func (m *GKEImage) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this g k e image based on context it is used
func (m *GKEImage) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *GKEImage) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *GKEImage) UnmarshalBinary(b []byte) error {
	var res GKEImage
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
