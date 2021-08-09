// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// AzureRouteTablesList AzureRouteTablesList is the object representing the route tables for vms in azure cloud provider
//
// swagger:model AzureRouteTablesList
type AzureRouteTablesList struct {

	// route tables
	RouteTables []string `json:"routeTables"`
}

// Validate validates this azure route tables list
func (m *AzureRouteTablesList) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this azure route tables list based on context it is used
func (m *AzureRouteTablesList) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AzureRouteTablesList) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AzureRouteTablesList) UnmarshalBinary(b []byte) error {
	var res AzureRouteTablesList
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
