// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ConstraintTemplateSpec ConstraintTemplateSpec is the object representing the gatekeeper constraint template spec and kubermatic related spec
//
// swagger:model ConstraintTemplateSpec
type ConstraintTemplateSpec struct {

	// targets
	Targets []*Target `json:"targets"`

	// crd
	Crd *CRD `json:"crd,omitempty"`

	// selector
	Selector *ConstraintTemplateSelector `json:"selector,omitempty"`
}

// Validate validates this constraint template spec
func (m *ConstraintTemplateSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateTargets(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateCrd(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateSelector(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ConstraintTemplateSpec) validateTargets(formats strfmt.Registry) error {

	if swag.IsZero(m.Targets) { // not required
		return nil
	}

	for i := 0; i < len(m.Targets); i++ {
		if swag.IsZero(m.Targets[i]) { // not required
			continue
		}

		if m.Targets[i] != nil {
			if err := m.Targets[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("targets" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *ConstraintTemplateSpec) validateCrd(formats strfmt.Registry) error {

	if swag.IsZero(m.Crd) { // not required
		return nil
	}

	if m.Crd != nil {
		if err := m.Crd.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("crd")
			}
			return err
		}
	}

	return nil
}

func (m *ConstraintTemplateSpec) validateSelector(formats strfmt.Registry) error {

	if swag.IsZero(m.Selector) { // not required
		return nil
	}

	if m.Selector != nil {
		if err := m.Selector.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("selector")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ConstraintTemplateSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ConstraintTemplateSpec) UnmarshalBinary(b []byte) error {
	var res ConstraintTemplateSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
