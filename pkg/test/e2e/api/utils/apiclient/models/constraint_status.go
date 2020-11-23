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

// ConstraintStatus ConstraintStatus represents a constraint status which holds audit info
//
// swagger:model ConstraintStatus
type ConstraintStatus struct {

	// audit timestamp
	AuditTimestamp string `json:"auditTimestamp,omitempty"`

	// enforcement
	Enforcement string `json:"enforcement,omitempty"`

	// violations
	Violations []*Violation `json:"violations"`
}

// Validate validates this constraint status
func (m *ConstraintStatus) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateViolations(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ConstraintStatus) validateViolations(formats strfmt.Registry) error {

	if swag.IsZero(m.Violations) { // not required
		return nil
	}

	for i := 0; i < len(m.Violations); i++ {
		if swag.IsZero(m.Violations[i]) { // not required
			continue
		}

		if m.Violations[i] != nil {
			if err := m.Violations[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("violations" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *ConstraintStatus) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ConstraintStatus) UnmarshalBinary(b []byte) error {
	var res ConstraintStatus
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
