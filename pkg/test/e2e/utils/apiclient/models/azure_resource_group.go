// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// AzureResourceGroup AzureResourceGroup represents an object of Azure ResourceGroup information.
//
// swagger:model AzureResourceGroup
type AzureResourceGroup struct {

	// The name of the resource group.
	Name string `json:"name,omitempty"`
}

// Validate validates this azure resource group
func (m *AzureResourceGroup) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this azure resource group based on context it is used
func (m *AzureResourceGroup) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AzureResourceGroup) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AzureResourceGroup) UnmarshalBinary(b []byte) error {
	var res AzureResourceGroup
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
