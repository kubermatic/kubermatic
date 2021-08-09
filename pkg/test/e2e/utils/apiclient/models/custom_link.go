// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// CustomLink custom link
//
// swagger:model CustomLink
type CustomLink struct {

	// icon
	Icon string `json:"icon,omitempty"`

	// label
	Label string `json:"label,omitempty"`

	// location
	Location string `json:"location,omitempty"`

	// URL
	URL string `json:"url,omitempty"`
}

// Validate validates this custom link
func (m *CustomLink) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this custom link based on context it is used
func (m *CustomLink) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *CustomLink) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *CustomLink) UnmarshalBinary(b []byte) error {
	var res CustomLink
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
