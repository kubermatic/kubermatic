// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// SSHKeySpec SSHKeySpec represents the details of a ssh key
//
// swagger:model SSHKeySpec
type SSHKeySpec struct {

	// fingerprint
	Fingerprint string `json:"fingerprint,omitempty"`

	// public key
	PublicKey string `json:"publicKey,omitempty"`
}

// Validate validates this SSH key spec
func (m *SSHKeySpec) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *SSHKeySpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *SSHKeySpec) UnmarshalBinary(b []byte) error {
	var res SSHKeySpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
