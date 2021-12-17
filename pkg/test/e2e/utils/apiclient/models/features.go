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

// Features features
//
// swagger:model Features
type Features struct {

	// acpi
	Acpi *FeatureState `json:"acpi,omitempty"`

	// apic
	Apic *FeatureAPIC `json:"apic,omitempty"`

	// hyperv
	Hyperv *FeatureHyperv `json:"hyperv,omitempty"`

	// kvm
	Kvm *FeatureKVM `json:"kvm,omitempty"`

	// pvspinlock
	Pvspinlock *FeatureState `json:"pvspinlock,omitempty"`

	// smm
	Smm *FeatureState `json:"smm,omitempty"`
}

// Validate validates this features
func (m *Features) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAcpi(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateApic(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateHyperv(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateKvm(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePvspinlock(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateSmm(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Features) validateAcpi(formats strfmt.Registry) error {
	if swag.IsZero(m.Acpi) { // not required
		return nil
	}

	if m.Acpi != nil {
		if err := m.Acpi.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("acpi")
			}
			return err
		}
	}

	return nil
}

func (m *Features) validateApic(formats strfmt.Registry) error {
	if swag.IsZero(m.Apic) { // not required
		return nil
	}

	if m.Apic != nil {
		if err := m.Apic.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("apic")
			}
			return err
		}
	}

	return nil
}

func (m *Features) validateHyperv(formats strfmt.Registry) error {
	if swag.IsZero(m.Hyperv) { // not required
		return nil
	}

	if m.Hyperv != nil {
		if err := m.Hyperv.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("hyperv")
			}
			return err
		}
	}

	return nil
}

func (m *Features) validateKvm(formats strfmt.Registry) error {
	if swag.IsZero(m.Kvm) { // not required
		return nil
	}

	if m.Kvm != nil {
		if err := m.Kvm.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kvm")
			}
			return err
		}
	}

	return nil
}

func (m *Features) validatePvspinlock(formats strfmt.Registry) error {
	if swag.IsZero(m.Pvspinlock) { // not required
		return nil
	}

	if m.Pvspinlock != nil {
		if err := m.Pvspinlock.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("pvspinlock")
			}
			return err
		}
	}

	return nil
}

func (m *Features) validateSmm(formats strfmt.Registry) error {
	if swag.IsZero(m.Smm) { // not required
		return nil
	}

	if m.Smm != nil {
		if err := m.Smm.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("smm")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this features based on the context it is used
func (m *Features) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateAcpi(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateApic(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateHyperv(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateKvm(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidatePvspinlock(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateSmm(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Features) contextValidateAcpi(ctx context.Context, formats strfmt.Registry) error {

	if m.Acpi != nil {
		if err := m.Acpi.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("acpi")
			}
			return err
		}
	}

	return nil
}

func (m *Features) contextValidateApic(ctx context.Context, formats strfmt.Registry) error {

	if m.Apic != nil {
		if err := m.Apic.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("apic")
			}
			return err
		}
	}

	return nil
}

func (m *Features) contextValidateHyperv(ctx context.Context, formats strfmt.Registry) error {

	if m.Hyperv != nil {
		if err := m.Hyperv.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("hyperv")
			}
			return err
		}
	}

	return nil
}

func (m *Features) contextValidateKvm(ctx context.Context, formats strfmt.Registry) error {

	if m.Kvm != nil {
		if err := m.Kvm.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kvm")
			}
			return err
		}
	}

	return nil
}

func (m *Features) contextValidatePvspinlock(ctx context.Context, formats strfmt.Registry) error {

	if m.Pvspinlock != nil {
		if err := m.Pvspinlock.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("pvspinlock")
			}
			return err
		}
	}

	return nil
}

func (m *Features) contextValidateSmm(ctx context.Context, formats strfmt.Registry) error {

	if m.Smm != nil {
		if err := m.Smm.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("smm")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *Features) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Features) UnmarshalBinary(b []byte) error {
	var res Features
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
