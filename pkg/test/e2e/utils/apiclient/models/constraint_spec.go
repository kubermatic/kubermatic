// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ConstraintSpec ConstraintSpec specifies the data for the constraint.
//
// swagger:model ConstraintSpec
type ConstraintSpec struct {

	// ConstraintType specifies the type of gatekeeper constraint that the constraint applies to
	ConstraintType string `json:"constraintType,omitempty"`

	// Disabled  is the flag for disabling OPA constraints
	Disabled bool `json:"disabled,omitempty"`

	// match
	Match *Match `json:"match,omitempty"`

	// parameters
	Parameters Parameters `json:"parameters,omitempty"`

	// selector
	Selector *ConstraintSelector `json:"selector,omitempty"`
}

// Validate validates this constraint spec
func (m *ConstraintSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateMatch(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateParameters(formats); err != nil {
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

func (m *ConstraintSpec) validateMatch(formats strfmt.Registry) error {
	if swag.IsZero(m.Match) { // not required
		return nil
	}

	if m.Match != nil {
		if err := m.Match.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("match")
			}
			return err
		}
	}

	return nil
}

func (m *ConstraintSpec) validateParameters(formats strfmt.Registry) error {
	if swag.IsZero(m.Parameters) { // not required
		return nil
	}

	if m.Parameters != nil {
		if err := m.Parameters.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("parameters")
			}
			return err
		}
	}

	return nil
}

func (m *ConstraintSpec) validateSelector(formats strfmt.Registry) error {
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

// ContextValidate validate this constraint spec based on the context it is used
func (m *ConstraintSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateMatch(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateParameters(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateSelector(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ConstraintSpec) contextValidateMatch(ctx context.Context, formats strfmt.Registry) error {

	if m.Match != nil {
		if err := m.Match.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("match")
			}
			return err
		}
	}

	return nil
}

func (m *ConstraintSpec) contextValidateParameters(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Parameters.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("parameters")
		}
		return err
	}

	return nil
}

func (m *ConstraintSpec) contextValidateSelector(ctx context.Context, formats strfmt.Registry) error {

	if m.Selector != nil {
		if err := m.Selector.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("selector")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ConstraintSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ConstraintSpec) UnmarshalBinary(b []byte) error {
	var res ConstraintSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
