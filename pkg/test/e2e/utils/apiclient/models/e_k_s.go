// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// EKS e k s
//
// swagger:model EKS
type EKS struct {

	// access key ID
	AccessKeyID string `json:"accessKeyId,omitempty"`

	// datacenter
	Datacenter string `json:"datacenter,omitempty"`

	// enabled
	Enabled bool `json:"enabled,omitempty"`

	// region
	Region string `json:"region,omitempty"`

	// secret access key
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
}

// Validate validates this e k s
func (m *EKS) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this e k s based on context it is used
func (m *EKS) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *EKS) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *EKS) UnmarshalBinary(b []byte) error {
	var res EKS
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
