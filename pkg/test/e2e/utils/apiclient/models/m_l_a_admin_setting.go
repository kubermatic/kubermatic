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

// MLAAdminSetting MLAAdminSetting represents an object holding admin setting options for user cluster MLA (Monitoring, Logging and Alerting).
//
// swagger:model MLAAdminSetting
type MLAAdminSetting struct {

	// logging rate limits
	LoggingRateLimits *LoggingRateLimitSettings `json:"loggingRateLimits,omitempty"`

	// monitoring rate limits
	MonitoringRateLimits *MonitoringRateLimitSettings `json:"monitoringRateLimits,omitempty"`
}

// Validate validates this m l a admin setting
func (m *MLAAdminSetting) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateLoggingRateLimits(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMonitoringRateLimits(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *MLAAdminSetting) validateLoggingRateLimits(formats strfmt.Registry) error {
	if swag.IsZero(m.LoggingRateLimits) { // not required
		return nil
	}

	if m.LoggingRateLimits != nil {
		if err := m.LoggingRateLimits.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("loggingRateLimits")
			}
			return err
		}
	}

	return nil
}

func (m *MLAAdminSetting) validateMonitoringRateLimits(formats strfmt.Registry) error {
	if swag.IsZero(m.MonitoringRateLimits) { // not required
		return nil
	}

	if m.MonitoringRateLimits != nil {
		if err := m.MonitoringRateLimits.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("monitoringRateLimits")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this m l a admin setting based on the context it is used
func (m *MLAAdminSetting) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateLoggingRateLimits(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateMonitoringRateLimits(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *MLAAdminSetting) contextValidateLoggingRateLimits(ctx context.Context, formats strfmt.Registry) error {

	if m.LoggingRateLimits != nil {
		if err := m.LoggingRateLimits.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("loggingRateLimits")
			}
			return err
		}
	}

	return nil
}

func (m *MLAAdminSetting) contextValidateMonitoringRateLimits(ctx context.Context, formats strfmt.Registry) error {

	if m.MonitoringRateLimits != nil {
		if err := m.MonitoringRateLimits.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("monitoringRateLimits")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *MLAAdminSetting) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *MLAAdminSetting) UnmarshalBinary(b []byte) error {
	var res MLAAdminSetting
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
