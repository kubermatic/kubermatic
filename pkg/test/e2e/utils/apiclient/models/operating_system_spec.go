// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// OperatingSystemSpec OperatingSystemSpec represents the collection of os specific settings. Only one must be set at a time.
//
// swagger:model OperatingSystemSpec
type OperatingSystemSpec struct {

	// centos
	Centos *CentOSSpec `json:"centos,omitempty"`

	// flatcar
	Flatcar *FlatcarSpec `json:"flatcar,omitempty"`

	// rhel
	Rhel *RHELSpec `json:"rhel,omitempty"`

	// sles
	Sles *SLESSpec `json:"sles,omitempty"`

	// ubuntu
	Ubuntu *UbuntuSpec `json:"ubuntu,omitempty"`
}

// Validate validates this operating system spec
func (m *OperatingSystemSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCentos(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateFlatcar(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRhel(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateSles(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUbuntu(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *OperatingSystemSpec) validateCentos(formats strfmt.Registry) error {

	if swag.IsZero(m.Centos) { // not required
		return nil
	}

	if m.Centos != nil {
		if err := m.Centos.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("centos")
			}
			return err
		}
	}

	return nil
}

func (m *OperatingSystemSpec) validateFlatcar(formats strfmt.Registry) error {

	if swag.IsZero(m.Flatcar) { // not required
		return nil
	}

	if m.Flatcar != nil {
		if err := m.Flatcar.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("flatcar")
			}
			return err
		}
	}

	return nil
}

func (m *OperatingSystemSpec) validateRhel(formats strfmt.Registry) error {

	if swag.IsZero(m.Rhel) { // not required
		return nil
	}

	if m.Rhel != nil {
		if err := m.Rhel.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("rhel")
			}
			return err
		}
	}

	return nil
}

func (m *OperatingSystemSpec) validateSles(formats strfmt.Registry) error {

	if swag.IsZero(m.Sles) { // not required
		return nil
	}

	if m.Sles != nil {
		if err := m.Sles.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("sles")
			}
			return err
		}
	}

	return nil
}

func (m *OperatingSystemSpec) validateUbuntu(formats strfmt.Registry) error {

	if swag.IsZero(m.Ubuntu) { // not required
		return nil
	}

	if m.Ubuntu != nil {
		if err := m.Ubuntu.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ubuntu")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *OperatingSystemSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *OperatingSystemSpec) UnmarshalBinary(b []byte) error {
	var res OperatingSystemSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
