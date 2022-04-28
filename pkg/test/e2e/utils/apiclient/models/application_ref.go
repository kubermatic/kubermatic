// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ApplicationRef application ref
//
// swagger:model ApplicationRef
type ApplicationRef struct {

	// Name of the Application
	Name string `json:"name,omitempty"`

	// version
	Version Version `json:"version,omitempty"`
}

// Validate validates this application ref
func (m *ApplicationRef) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this application ref based on context it is used
func (m *ApplicationRef) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *ApplicationRef) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ApplicationRef) UnmarshalBinary(b []byte) error {
	var res ApplicationRef
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
