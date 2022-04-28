// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// MeteringConfiguration MeteringConfiguration contains all the configuration for the metering tool.
//
// swagger:model MeteringConfiguration
type MeteringConfiguration struct {

	// enabled
	Enabled bool `json:"enabled,omitempty"`

	// ReportConfigurations is a map of report configuration definitions.
	ReportConfigurations map[string]MeteringReportConfiguration `json:"reports,omitempty"`

	// StorageClassName is the name of the storage class that the metering tool uses to save processed files before
	// exporting it to s3 bucket. Default value is kubermatic-fast.
	StorageClassName string `json:"storageClassName,omitempty"`

	// StorageSize is the size of the storage class. Default value is 100Gi.
	StorageSize string `json:"storageSize,omitempty"`
}

// Validate validates this metering configuration
func (m *MeteringConfiguration) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateReportConfigurations(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *MeteringConfiguration) validateReportConfigurations(formats strfmt.Registry) error {
	if swag.IsZero(m.ReportConfigurations) { // not required
		return nil
	}

	for k := range m.ReportConfigurations {

		if err := validate.Required("reports"+"."+k, "body", m.ReportConfigurations[k]); err != nil {
			return err
		}
		if val, ok := m.ReportConfigurations[k]; ok {
			if err := val.Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("reports" + "." + k)
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("reports" + "." + k)
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this metering configuration based on the context it is used
func (m *MeteringConfiguration) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateReportConfigurations(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *MeteringConfiguration) contextValidateReportConfigurations(ctx context.Context, formats strfmt.Registry) error {

	for k := range m.ReportConfigurations {

		if val, ok := m.ReportConfigurations[k]; ok {
			if err := val.ContextValidate(ctx, formats); err != nil {
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *MeteringConfiguration) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *MeteringConfiguration) UnmarshalBinary(b []byte) error {
	var res MeteringConfiguration
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
