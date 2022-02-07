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

// FloppyTarget floppy target
//
// swagger:model FloppyTarget
type FloppyTarget struct {

	// ReadOnly.
	// Defaults to false.
	ReadOnly bool `json:"readonly,omitempty"`

	// tray
	Tray TrayState `json:"tray,omitempty"`
}

// Validate validates this floppy target
func (m *FloppyTarget) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateTray(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *FloppyTarget) validateTray(formats strfmt.Registry) error {
	if swag.IsZero(m.Tray) { // not required
		return nil
	}

	if err := m.Tray.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("tray")
		}
		return err
	}

	return nil
}

// ContextValidate validate this floppy target based on the context it is used
func (m *FloppyTarget) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateTray(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *FloppyTarget) contextValidateTray(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Tray.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("tray")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *FloppyTarget) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *FloppyTarget) UnmarshalBinary(b []byte) error {
	var res FloppyTarget
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
