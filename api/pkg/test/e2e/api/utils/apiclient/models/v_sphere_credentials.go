// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// VSphereCredentials VSphereCredentials credentials represents a credential for accessing vSphere
//
// swagger:model VSphereCredentials
type VSphereCredentials struct {

	// password
	Password string `json:"password,omitempty"`

	// username
	Username string `json:"username,omitempty"`
}

// Validate validates this v sphere credentials
func (m *VSphereCredentials) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *VSphereCredentials) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *VSphereCredentials) UnmarshalBinary(b []byte) error {
	var res VSphereCredentials
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
