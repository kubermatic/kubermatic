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

// Firmware firmware
//
// swagger:model Firmware
type Firmware struct {

	// The system-serial-number in SMBIOS
	Serial string `json:"serial,omitempty"`

	// bootloader
	Bootloader *Bootloader `json:"bootloader,omitempty"`

	// kernel boot
	KernelBoot *KernelBoot `json:"kernelBoot,omitempty"`

	// uuid
	UUID UID `json:"uuid,omitempty"`
}

// Validate validates this firmware
func (m *Firmware) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateBootloader(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateKernelBoot(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUUID(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Firmware) validateBootloader(formats strfmt.Registry) error {
	if swag.IsZero(m.Bootloader) { // not required
		return nil
	}

	if m.Bootloader != nil {
		if err := m.Bootloader.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("bootloader")
			}
			return err
		}
	}

	return nil
}

func (m *Firmware) validateKernelBoot(formats strfmt.Registry) error {
	if swag.IsZero(m.KernelBoot) { // not required
		return nil
	}

	if m.KernelBoot != nil {
		if err := m.KernelBoot.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kernelBoot")
			}
			return err
		}
	}

	return nil
}

func (m *Firmware) validateUUID(formats strfmt.Registry) error {
	if swag.IsZero(m.UUID) { // not required
		return nil
	}

	if err := m.UUID.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("uuid")
		}
		return err
	}

	return nil
}

// ContextValidate validate this firmware based on the context it is used
func (m *Firmware) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateBootloader(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateKernelBoot(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateUUID(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Firmware) contextValidateBootloader(ctx context.Context, formats strfmt.Registry) error {

	if m.Bootloader != nil {
		if err := m.Bootloader.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("bootloader")
			}
			return err
		}
	}

	return nil
}

func (m *Firmware) contextValidateKernelBoot(ctx context.Context, formats strfmt.Registry) error {

	if m.KernelBoot != nil {
		if err := m.KernelBoot.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kernelBoot")
			}
			return err
		}
	}

	return nil
}

func (m *Firmware) contextValidateUUID(ctx context.Context, formats strfmt.Registry) error {

	if err := m.UUID.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("uuid")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *Firmware) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Firmware) UnmarshalBinary(b []byte) error {
	var res Firmware
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
