// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// OpenstackSubnet OpenstackSubnet is the object representing a openstack subnet.
//
// swagger:model OpenstackSubnet
type OpenstackSubnet struct {

	// Id uniquely identifies the subnet
	ID string `json:"id,omitempty"`

	// Name is human-readable name for the subnet
	Name string `json:"name,omitempty"`
}

// Validate validates this openstack subnet
func (m *OpenstackSubnet) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this openstack subnet based on context it is used
func (m *OpenstackSubnet) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *OpenstackSubnet) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *OpenstackSubnet) UnmarshalBinary(b []byte) error {
	var res OpenstackSubnet
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
