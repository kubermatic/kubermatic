// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// CreateCRDError CreateCRDError represents a single error caught during parsing, compiling, etc.
//
// swagger:model CreateCRDError
type CreateCRDError struct {

	// code
	Code string `json:"code,omitempty"`

	// location
	Location string `json:"location,omitempty"`

	// message
	Message string `json:"message,omitempty"`
}

// Validate validates this create c r d error
func (m *CreateCRDError) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *CreateCRDError) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *CreateCRDError) UnmarshalBinary(b []byte) error {
	var res CreateCRDError
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
