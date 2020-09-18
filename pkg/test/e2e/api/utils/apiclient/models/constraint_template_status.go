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

// ConstraintTemplateStatus ConstraintTemplateStatus defines the observed state of ConstraintTemplate
//
// swagger:model ConstraintTemplateStatus
type ConstraintTemplateStatus struct {

	// by pod
	ByPod []*ByPodStatus `json:"byPod"`

	// created
	Created bool `json:"created,omitempty"`
}

// Validate validates this constraint template status
func (m *ConstraintTemplateStatus) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateByPod(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ConstraintTemplateStatus) validateByPod(formats strfmt.Registry) error {

	if swag.IsZero(m.ByPod) { // not required
		return nil
	}

	for i := 0; i < len(m.ByPod); i++ {
		if swag.IsZero(m.ByPod[i]) { // not required
			continue
		}

		if m.ByPod[i] != nil {
			if err := m.ByPod[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("byPod" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *ConstraintTemplateStatus) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ConstraintTemplateStatus) UnmarshalBinary(b []byte) error {
	var res ConstraintTemplateStatus
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
