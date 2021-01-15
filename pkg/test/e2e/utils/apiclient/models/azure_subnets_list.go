// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// AzureSubnetsList AzureSubnetsList is the object representing the subnets for vms in azure cloud provider
//
// swagger:model AzureSubnetsList
type AzureSubnetsList struct {

	// subnets
	Subnets []string `json:"subnets"`
}

// Validate validates this azure subnets list
func (m *AzureSubnetsList) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AzureSubnetsList) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AzureSubnetsList) UnmarshalBinary(b []byte) error {
	var res AzureSubnetsList
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
