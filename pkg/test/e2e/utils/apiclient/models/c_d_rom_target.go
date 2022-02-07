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

// CDRomTarget c d rom target
//
// swagger:model CDRomTarget
type CDRomTarget struct {

	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi.
	Bus string `json:"bus,omitempty"`

	// ReadOnly.
	// Defaults to true.
	ReadOnly bool `json:"readonly,omitempty"`

	// tray
	Tray TrayState `json:"tray,omitempty"`
}

// Validate validates this c d rom target
func (m *CDRomTarget) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateTray(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *CDRomTarget) validateTray(formats strfmt.Registry) error {
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

// ContextValidate validate this c d rom target based on the context it is used
func (m *CDRomTarget) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateTray(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *CDRomTarget) contextValidateTray(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Tray.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("tray")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *CDRomTarget) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *CDRomTarget) UnmarshalBinary(b []byte) error {
	var res CDRomTarget
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
