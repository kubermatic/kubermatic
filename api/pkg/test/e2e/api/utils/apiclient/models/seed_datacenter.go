// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// SeedDatacenter seed datacenter
//
// swagger:model SeedDatacenter
type SeedDatacenter struct {

	// Optional: Country of the seed as ISO-3166 two-letter code, e.g. DE or UK.
	// For informational purposes in the Kubermatic dashboard only.
	Country string `json:"country,omitempty"`

	// Optional: Detailed location of the cluster, like "Hamburg" or "Datacenter 7".
	// For informational purposes in the Kubermatic dashboard only.
	Location string `json:"location,omitempty"`

	// node
	Node *NodeSettings `json:"node,omitempty"`

	// spec
	Spec *SeedDatacenterSpec `json:"spec,omitempty"`
}

// Validate validates this seed datacenter
func (m *SeedDatacenter) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateNode(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateSpec(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *SeedDatacenter) validateNode(formats strfmt.Registry) error {

	if swag.IsZero(m.Node) { // not required
		return nil
	}

	if m.Node != nil {
		if err := m.Node.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("node")
			}
			return err
		}
	}

	return nil
}

func (m *SeedDatacenter) validateSpec(formats strfmt.Registry) error {

	if swag.IsZero(m.Spec) { // not required
		return nil
	}

	if m.Spec != nil {
		if err := m.Spec.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("spec")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *SeedDatacenter) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *SeedDatacenter) UnmarshalBinary(b []byte) error {
	var res SeedDatacenter
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
