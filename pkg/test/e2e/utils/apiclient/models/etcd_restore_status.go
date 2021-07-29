// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// EtcdRestoreStatus etcd restore status
//
// swagger:model EtcdRestoreStatus
type EtcdRestoreStatus struct {

	// phase
	Phase EtcdRestorePhase `json:"phase,omitempty"`

	// restore time
	// Format: date-time
	RestoreTime Time `json:"restoreTime,omitempty"`
}

// Validate validates this etcd restore status
func (m *EtcdRestoreStatus) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validatePhase(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRestoreTime(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *EtcdRestoreStatus) validatePhase(formats strfmt.Registry) error {

	if swag.IsZero(m.Phase) { // not required
		return nil
	}

	if err := m.Phase.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("phase")
		}
		return err
	}

	return nil
}

func (m *EtcdRestoreStatus) validateRestoreTime(formats strfmt.Registry) error {

	if swag.IsZero(m.RestoreTime) { // not required
		return nil
	}

	if err := m.RestoreTime.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("restoreTime")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *EtcdRestoreStatus) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *EtcdRestoreStatus) UnmarshalBinary(b []byte) error {
	var res EtcdRestoreStatus
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
