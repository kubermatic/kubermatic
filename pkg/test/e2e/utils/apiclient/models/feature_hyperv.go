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

// FeatureHyperv Hyperv specific features.
//
// swagger:model FeatureHyperv
type FeatureHyperv struct {

	// evmcs
	Evmcs *FeatureState `json:"evmcs,omitempty"`

	// frequencies
	Frequencies *FeatureState `json:"frequencies,omitempty"`

	// ipi
	Ipi *FeatureState `json:"ipi,omitempty"`

	// reenlightenment
	Reenlightenment *FeatureState `json:"reenlightenment,omitempty"`

	// relaxed
	Relaxed *FeatureState `json:"relaxed,omitempty"`

	// reset
	Reset *FeatureState `json:"reset,omitempty"`

	// runtime
	Runtime *FeatureState `json:"runtime,omitempty"`

	// spinlocks
	Spinlocks *FeatureSpinlocks `json:"spinlocks,omitempty"`

	// synic
	Synic *FeatureState `json:"synic,omitempty"`

	// synictimer
	Synictimer *SyNICTimer `json:"synictimer,omitempty"`

	// tlbflush
	Tlbflush *FeatureState `json:"tlbflush,omitempty"`

	// vapic
	Vapic *FeatureState `json:"vapic,omitempty"`

	// vendorid
	Vendorid *FeatureVendorID `json:"vendorid,omitempty"`

	// vpindex
	Vpindex *FeatureState `json:"vpindex,omitempty"`
}

// Validate validates this feature hyperv
func (m *FeatureHyperv) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateEvmcs(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateFrequencies(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateIpi(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateReenlightenment(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRelaxed(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateReset(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateRuntime(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateSpinlocks(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateSynic(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateSynictimer(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateTlbflush(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVapic(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVendorid(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVpindex(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *FeatureHyperv) validateEvmcs(formats strfmt.Registry) error {
	if swag.IsZero(m.Evmcs) { // not required
		return nil
	}

	if m.Evmcs != nil {
		if err := m.Evmcs.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("evmcs")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateFrequencies(formats strfmt.Registry) error {
	if swag.IsZero(m.Frequencies) { // not required
		return nil
	}

	if m.Frequencies != nil {
		if err := m.Frequencies.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("frequencies")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateIpi(formats strfmt.Registry) error {
	if swag.IsZero(m.Ipi) { // not required
		return nil
	}

	if m.Ipi != nil {
		if err := m.Ipi.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ipi")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateReenlightenment(formats strfmt.Registry) error {
	if swag.IsZero(m.Reenlightenment) { // not required
		return nil
	}

	if m.Reenlightenment != nil {
		if err := m.Reenlightenment.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("reenlightenment")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateRelaxed(formats strfmt.Registry) error {
	if swag.IsZero(m.Relaxed) { // not required
		return nil
	}

	if m.Relaxed != nil {
		if err := m.Relaxed.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("relaxed")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateReset(formats strfmt.Registry) error {
	if swag.IsZero(m.Reset) { // not required
		return nil
	}

	if m.Reset != nil {
		if err := m.Reset.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("reset")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateRuntime(formats strfmt.Registry) error {
	if swag.IsZero(m.Runtime) { // not required
		return nil
	}

	if m.Runtime != nil {
		if err := m.Runtime.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("runtime")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateSpinlocks(formats strfmt.Registry) error {
	if swag.IsZero(m.Spinlocks) { // not required
		return nil
	}

	if m.Spinlocks != nil {
		if err := m.Spinlocks.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("spinlocks")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateSynic(formats strfmt.Registry) error {
	if swag.IsZero(m.Synic) { // not required
		return nil
	}

	if m.Synic != nil {
		if err := m.Synic.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("synic")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateSynictimer(formats strfmt.Registry) error {
	if swag.IsZero(m.Synictimer) { // not required
		return nil
	}

	if m.Synictimer != nil {
		if err := m.Synictimer.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("synictimer")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateTlbflush(formats strfmt.Registry) error {
	if swag.IsZero(m.Tlbflush) { // not required
		return nil
	}

	if m.Tlbflush != nil {
		if err := m.Tlbflush.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("tlbflush")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateVapic(formats strfmt.Registry) error {
	if swag.IsZero(m.Vapic) { // not required
		return nil
	}

	if m.Vapic != nil {
		if err := m.Vapic.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vapic")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateVendorid(formats strfmt.Registry) error {
	if swag.IsZero(m.Vendorid) { // not required
		return nil
	}

	if m.Vendorid != nil {
		if err := m.Vendorid.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vendorid")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) validateVpindex(formats strfmt.Registry) error {
	if swag.IsZero(m.Vpindex) { // not required
		return nil
	}

	if m.Vpindex != nil {
		if err := m.Vpindex.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vpindex")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this feature hyperv based on the context it is used
func (m *FeatureHyperv) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateEvmcs(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateFrequencies(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateIpi(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateReenlightenment(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateRelaxed(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateReset(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateRuntime(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateSpinlocks(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateSynic(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateSynictimer(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateTlbflush(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateVapic(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateVendorid(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateVpindex(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *FeatureHyperv) contextValidateEvmcs(ctx context.Context, formats strfmt.Registry) error {

	if m.Evmcs != nil {
		if err := m.Evmcs.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("evmcs")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateFrequencies(ctx context.Context, formats strfmt.Registry) error {

	if m.Frequencies != nil {
		if err := m.Frequencies.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("frequencies")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateIpi(ctx context.Context, formats strfmt.Registry) error {

	if m.Ipi != nil {
		if err := m.Ipi.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ipi")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateReenlightenment(ctx context.Context, formats strfmt.Registry) error {

	if m.Reenlightenment != nil {
		if err := m.Reenlightenment.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("reenlightenment")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateRelaxed(ctx context.Context, formats strfmt.Registry) error {

	if m.Relaxed != nil {
		if err := m.Relaxed.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("relaxed")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateReset(ctx context.Context, formats strfmt.Registry) error {

	if m.Reset != nil {
		if err := m.Reset.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("reset")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateRuntime(ctx context.Context, formats strfmt.Registry) error {

	if m.Runtime != nil {
		if err := m.Runtime.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("runtime")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateSpinlocks(ctx context.Context, formats strfmt.Registry) error {

	if m.Spinlocks != nil {
		if err := m.Spinlocks.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("spinlocks")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateSynic(ctx context.Context, formats strfmt.Registry) error {

	if m.Synic != nil {
		if err := m.Synic.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("synic")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateSynictimer(ctx context.Context, formats strfmt.Registry) error {

	if m.Synictimer != nil {
		if err := m.Synictimer.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("synictimer")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateTlbflush(ctx context.Context, formats strfmt.Registry) error {

	if m.Tlbflush != nil {
		if err := m.Tlbflush.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("tlbflush")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateVapic(ctx context.Context, formats strfmt.Registry) error {

	if m.Vapic != nil {
		if err := m.Vapic.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vapic")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateVendorid(ctx context.Context, formats strfmt.Registry) error {

	if m.Vendorid != nil {
		if err := m.Vendorid.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vendorid")
			}
			return err
		}
	}

	return nil
}

func (m *FeatureHyperv) contextValidateVpindex(ctx context.Context, formats strfmt.Registry) error {

	if m.Vpindex != nil {
		if err := m.Vpindex.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vpindex")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *FeatureHyperv) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *FeatureHyperv) UnmarshalBinary(b []byte) error {
	var res FeatureHyperv
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
