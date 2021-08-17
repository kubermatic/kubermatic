// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// DigitaloceanSizeList DigitaloceanSizeList represents a object of digitalocean sizes.
//
// swagger:model DigitaloceanSizeList
type DigitaloceanSizeList struct {

	// optimized
	Optimized []*DigitaloceanSize `json:"optimized"`

	// standard
	Standard []*DigitaloceanSize `json:"standard"`
}

// Validate validates this digitalocean size list
func (m *DigitaloceanSizeList) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateOptimized(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateStandard(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *DigitaloceanSizeList) validateOptimized(formats strfmt.Registry) error {
	if swag.IsZero(m.Optimized) { // not required
		return nil
	}

	for i := 0; i < len(m.Optimized); i++ {
		if swag.IsZero(m.Optimized[i]) { // not required
			continue
		}

		if m.Optimized[i] != nil {
			if err := m.Optimized[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("optimized" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *DigitaloceanSizeList) validateStandard(formats strfmt.Registry) error {
	if swag.IsZero(m.Standard) { // not required
		return nil
	}

	for i := 0; i < len(m.Standard); i++ {
		if swag.IsZero(m.Standard[i]) { // not required
			continue
		}

		if m.Standard[i] != nil {
			if err := m.Standard[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("standard" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this digitalocean size list based on the context it is used
func (m *DigitaloceanSizeList) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateOptimized(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateStandard(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *DigitaloceanSizeList) contextValidateOptimized(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.Optimized); i++ {

		if m.Optimized[i] != nil {
			if err := m.Optimized[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("optimized" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *DigitaloceanSizeList) contextValidateStandard(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.Standard); i++ {

		if m.Standard[i] != nil {
			if err := m.Standard[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("standard" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *DigitaloceanSizeList) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *DigitaloceanSizeList) UnmarshalBinary(b []byte) error {
	var res DigitaloceanSizeList
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
