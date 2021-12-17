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

// Clock Represents the clock and timers of a vmi.
//
// +kubebuilder:pruning:PreserveUnknownFields
//
// swagger:model Clock
type Clock struct {

	// timer
	Timer *Timer `json:"timer,omitempty"`

	// timezone
	Timezone ClockOffsetTimezone `json:"timezone,omitempty"`

	// utc
	Utc *ClockOffsetUTC `json:"utc,omitempty"`
}

// Validate validates this clock
func (m *Clock) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateTimer(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateTimezone(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUtc(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Clock) validateTimer(formats strfmt.Registry) error {
	if swag.IsZero(m.Timer) { // not required
		return nil
	}

	if m.Timer != nil {
		if err := m.Timer.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("timer")
			}
			return err
		}
	}

	return nil
}

func (m *Clock) validateTimezone(formats strfmt.Registry) error {
	if swag.IsZero(m.Timezone) { // not required
		return nil
	}

	if err := m.Timezone.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("timezone")
		}
		return err
	}

	return nil
}

func (m *Clock) validateUtc(formats strfmt.Registry) error {
	if swag.IsZero(m.Utc) { // not required
		return nil
	}

	if m.Utc != nil {
		if err := m.Utc.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("utc")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this clock based on the context it is used
func (m *Clock) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateTimer(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateTimezone(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateUtc(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Clock) contextValidateTimer(ctx context.Context, formats strfmt.Registry) error {

	if m.Timer != nil {
		if err := m.Timer.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("timer")
			}
			return err
		}
	}

	return nil
}

func (m *Clock) contextValidateTimezone(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Timezone.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("timezone")
		}
		return err
	}

	return nil
}

func (m *Clock) contextValidateUtc(ctx context.Context, formats strfmt.Registry) error {

	if m.Utc != nil {
		if err := m.Utc.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("utc")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *Clock) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Clock) UnmarshalBinary(b []byte) error {
	var res Clock
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
