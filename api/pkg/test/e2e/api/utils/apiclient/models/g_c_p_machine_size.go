// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// GCPMachineSize GCPMachineSize represents a object of GCP machine size.
// swagger:model GCPMachineSize
type GCPMachineSize struct {

	// description
	Description string `json:"description,omitempty"`

	// memory
	Memory int64 `json:"memory,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// v c p us
	VCPUs int64 `json:"vcpus,omitempty"`
}

// Validate validates this g c p machine size
func (m *GCPMachineSize) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *GCPMachineSize) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *GCPMachineSize) UnmarshalBinary(b []byte) error {
	var res GCPMachineSize
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
