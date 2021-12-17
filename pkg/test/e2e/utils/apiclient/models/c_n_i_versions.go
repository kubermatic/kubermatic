// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// CNIVersions CNIVersions is a list of versions for a CNI Plugin
//
// swagger:model CNIVersions
type CNIVersions struct {

	// CNIPluginType represents the type of the CNI Plugin
	CNIPluginType string `json:"CNIPluginType,omitempty"`

	// Versions represents the list of the CNI Plugin versions that are supported
	Versions []string `json:"Versions"`
}

// Validate validates this c n i versions
func (m *CNIVersions) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this c n i versions based on context it is used
func (m *CNIVersions) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *CNIVersions) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *CNIVersions) UnmarshalBinary(b []byte) error {
	var res CNIVersions
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
