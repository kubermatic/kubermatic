// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/swag"
)

// PacketCloudSpec PacketCloudSpec specifies access data to a Packet cloud.
// swagger:model PacketCloudSpec
type PacketCloudSpec struct {

	// API key
	APIKey string `json:"apiKey,omitempty"`

	// billing cycle
	BillingCycle string `json:"billingCycle,omitempty"`

	// project ID
	ProjectID string `json:"projectID,omitempty"`
}

// Validate validates this packet cloud spec
func (m *PacketCloudSpec) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *PacketCloudSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *PacketCloudSpec) UnmarshalBinary(b []byte) error {
	var res PacketCloudSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
