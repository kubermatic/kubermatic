// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// AzureResourceGroupsList AzureResourceGroupsList is the object representing the resource groups for vms in azure cloud provider
//
// swagger:model AzureResourceGroupsList
type AzureResourceGroupsList struct {

	// resource groups
	ResourceGroups []string `json:"resourceGroups"`
}

// Validate validates this azure resource groups list
func (m *AzureResourceGroupsList) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this azure resource groups list based on context it is used
func (m *AzureResourceGroupsList) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AzureResourceGroupsList) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AzureResourceGroupsList) UnmarshalBinary(b []byte) error {
	var res AzureResourceGroupsList
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
