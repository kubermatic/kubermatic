// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// VSphereFolder VSphereFolder is the object representing a vsphere folder.
//
// swagger:model VSphereFolder
type VSphereFolder struct {

	// Path is the path of the folder
	Path string `json:"path,omitempty"`
}

// Validate validates this v sphere folder
func (m *VSphereFolder) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this v sphere folder based on context it is used
func (m *VSphereFolder) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *VSphereFolder) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *VSphereFolder) UnmarshalBinary(b []byte) error {
	var res VSphereFolder
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
