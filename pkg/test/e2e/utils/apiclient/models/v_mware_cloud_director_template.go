// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// VMwareCloudDirectorTemplate VMwareCloudDirectorTemplate represents a VMware Cloud Director template.
//
// swagger:model VMwareCloudDirectorTemplate
type VMwareCloudDirectorTemplate struct {

	// name
	Name string `json:"name,omitempty"`
}

// Validate validates this v mware cloud director template
func (m *VMwareCloudDirectorTemplate) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this v mware cloud director template based on context it is used
func (m *VMwareCloudDirectorTemplate) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *VMwareCloudDirectorTemplate) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *VMwareCloudDirectorTemplate) UnmarshalBinary(b []byte) error {
	var res VMwareCloudDirectorTemplate
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
