// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// PacketCPU PacketCPU represents an array of Packet CPUs. It is a part of PacketSize.
//
// swagger:model PacketCPU
type PacketCPU struct {

	// count
	Count int64 `json:"count,omitempty"`

	// type
	Type string `json:"type,omitempty"`
}

// Validate validates this packet CPU
func (m *PacketCPU) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this packet CPU based on context it is used
func (m *PacketCPU) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *PacketCPU) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *PacketCPU) UnmarshalBinary(b []byte) error {
	var res PacketCPU
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
