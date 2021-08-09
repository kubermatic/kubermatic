// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// GCPZone GCPZone represents a object of GCP zone.
//
// swagger:model GCPZone
type GCPZone struct {

	// name
	Name string `json:"name,omitempty"`
}

// Validate validates this g c p zone
func (m *GCPZone) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this g c p zone based on context it is used
func (m *GCPZone) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *GCPZone) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *GCPZone) UnmarshalBinary(b []byte) error {
	var res GCPZone
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
