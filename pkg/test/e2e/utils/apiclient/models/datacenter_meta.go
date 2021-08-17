// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// DatacenterMeta DatacenterMeta holds datacenter metadata information.
//
// swagger:model DatacenterMeta
type DatacenterMeta struct {

	// name
	Name string `json:"name,omitempty"`
}

// Validate validates this datacenter meta
func (m *DatacenterMeta) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this datacenter meta based on context it is used
func (m *DatacenterMeta) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *DatacenterMeta) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *DatacenterMeta) UnmarshalBinary(b []byte) error {
	var res DatacenterMeta
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
