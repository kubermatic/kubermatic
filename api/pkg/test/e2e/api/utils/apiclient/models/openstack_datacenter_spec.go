// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// OpenstackDatacenterSpec OpenstackDatacenterSpec specifies a generic bare metal datacenter.
//
// swagger:model OpenstackDatacenterSpec
type OpenstackDatacenterSpec struct {

	// auth URL
	AuthURL string `json:"auth_url,omitempty"`

	// availability zone
	AvailabilityZone string `json:"availability_zone,omitempty"`

	// enforce floating IP
	EnforceFloatingIP bool `json:"enforce_floating_ip,omitempty"`

	// region
	Region string `json:"region,omitempty"`

	// images
	Images ImageList `json:"images,omitempty"`
}

// Validate validates this openstack datacenter spec
func (m *OpenstackDatacenterSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateImages(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *OpenstackDatacenterSpec) validateImages(formats strfmt.Registry) error {

	if swag.IsZero(m.Images) { // not required
		return nil
	}

	if err := m.Images.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("images")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *OpenstackDatacenterSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *OpenstackDatacenterSpec) UnmarshalBinary(b []byte) error {
	var res OpenstackDatacenterSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
