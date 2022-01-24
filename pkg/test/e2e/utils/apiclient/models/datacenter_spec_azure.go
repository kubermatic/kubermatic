// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// DatacenterSpecAzure DatacenterSpecAzure describes an Azure cloud datacenter.
//
// swagger:model DatacenterSpecAzure
type DatacenterSpecAzure struct {

	// Region to use, for example "westeurope". A list of available regions can be
	// found at https://azure.microsoft.com/en-us/global-infrastructure/locations/
	Location string `json:"location,omitempty"`
}

// Validate validates this datacenter spec azure
func (m *DatacenterSpecAzure) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this datacenter spec azure based on context it is used
func (m *DatacenterSpecAzure) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *DatacenterSpecAzure) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *DatacenterSpecAzure) UnmarshalBinary(b []byte) error {
	var res DatacenterSpecAzure
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
