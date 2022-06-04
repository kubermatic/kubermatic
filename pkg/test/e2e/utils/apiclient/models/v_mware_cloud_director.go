// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// VMwareCloudDirector v mware cloud director
//
// swagger:model VMwareCloudDirector
type VMwareCloudDirector struct {

	// datacenter
	Datacenter string `json:"datacenter,omitempty"`

	// enabled
	Enabled bool `json:"enabled,omitempty"`

	// o v d c network
	OVDCNetwork string `json:"ovdcNetwork,omitempty"`

	// organization
	Organization string `json:"organization,omitempty"`

	// password
	Password string `json:"password,omitempty"`

	// username
	Username string `json:"username,omitempty"`

	// v d c
	VDC string `json:"vdc,omitempty"`
}

// Validate validates this v mware cloud director
func (m *VMwareCloudDirector) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this v mware cloud director based on context it is used
func (m *VMwareCloudDirector) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *VMwareCloudDirector) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *VMwareCloudDirector) UnmarshalBinary(b []byte) error {
	var res VMwareCloudDirector
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
