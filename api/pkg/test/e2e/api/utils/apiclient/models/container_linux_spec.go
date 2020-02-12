// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ContainerLinuxSpec ContainerLinuxSpec ubuntu linux specific settings
// swagger:model ContainerLinuxSpec
type ContainerLinuxSpec struct {

	// disable container linux auto-update feature
	DisableAutoUpdate bool `json:"disableAutoUpdate,omitempty"`
}

// Validate validates this container linux spec
func (m *ContainerLinuxSpec) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *ContainerLinuxSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ContainerLinuxSpec) UnmarshalBinary(b []byte) error {
	var res ContainerLinuxSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
