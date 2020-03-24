// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// AzureDatacenterSpec AzureDatacenterSpec specifies a datacenter of Azure.
//
// swagger:model AzureDatacenterSpec
type AzureDatacenterSpec struct {

	// location
	Location string `json:"location,omitempty"`
}

// Validate validates this azure datacenter spec
func (m *AzureDatacenterSpec) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AzureDatacenterSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AzureDatacenterSpec) UnmarshalBinary(b []byte) error {
	var res AzureDatacenterSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
