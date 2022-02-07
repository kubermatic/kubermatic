// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// CustomBlockSize CustomBlockSize represents the desired logical and physical block size for a VM disk.
//
// swagger:model CustomBlockSize
type CustomBlockSize struct {

	// logical
	Logical uint64 `json:"logical,omitempty"`

	// physical
	Physical uint64 `json:"physical,omitempty"`
}

// Validate validates this custom block size
func (m *CustomBlockSize) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this custom block size based on context it is used
func (m *CustomBlockSize) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *CustomBlockSize) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *CustomBlockSize) UnmarshalBinary(b []byte) error {
	var res CustomBlockSize
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
