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

// ConstraintSelector ConstraintSelector is the object holding the cluster selection filters.
//
// swagger:model ConstraintSelector
type ConstraintSelector struct {

	// Providers is a list of cloud providers to which the Constraint applies to. Empty means all providers are selected.
	Providers []string `json:"providers"`

	// label selector
	LabelSelector *LabelSelector `json:"labelSelector,omitempty"`
}

// Validate validates this constraint selector
func (m *ConstraintSelector) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateLabelSelector(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ConstraintSelector) validateLabelSelector(formats strfmt.Registry) error {
	if swag.IsZero(m.LabelSelector) { // not required
		return nil
	}

	if m.LabelSelector != nil {
		if err := m.LabelSelector.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("labelSelector")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("labelSelector")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this constraint selector based on the context it is used
func (m *ConstraintSelector) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateLabelSelector(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ConstraintSelector) contextValidateLabelSelector(ctx context.Context, formats strfmt.Registry) error {

	if m.LabelSelector != nil {
		if err := m.LabelSelector.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("labelSelector")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("labelSelector")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ConstraintSelector) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ConstraintSelector) UnmarshalBinary(b []byte) error {
	var res ConstraintSelector
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
