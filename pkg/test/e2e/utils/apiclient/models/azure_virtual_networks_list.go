// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// AzureVirtualNetworksList AzureVirtualNetworksList is the object representing the virtual network for vms in azure cloud provider
//
// swagger:model AzureVirtualNetworksList
type AzureVirtualNetworksList struct {

	// virtual networks
	VirtualNetworks []string `json:"virtualNetworks"`
}

// Validate validates this azure virtual networks list
func (m *AzureVirtualNetworksList) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AzureVirtualNetworksList) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AzureVirtualNetworksList) UnmarshalBinary(b []byte) error {
	var res AzureVirtualNetworksList
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
