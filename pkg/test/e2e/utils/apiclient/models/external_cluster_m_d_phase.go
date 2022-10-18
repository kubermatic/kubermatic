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

// ExternalClusterMDPhase ExternalClusterMDPhase defines the external cluster machinedeployment phase.
//
// swagger:model ExternalClusterMDPhase
type ExternalClusterMDPhase struct {

	// status message
	StatusMessage string `json:"statusMessage,omitempty"`

	// aks
	Aks *AKSMDPhase `json:"aks,omitempty"`

	// state
	State ExternalClusterMDState `json:"state,omitempty"`
}

// Validate validates this external cluster m d phase
func (m *ExternalClusterMDPhase) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAks(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateState(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ExternalClusterMDPhase) validateAks(formats strfmt.Registry) error {
	if swag.IsZero(m.Aks) { // not required
		return nil
	}

	if m.Aks != nil {
		if err := m.Aks.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("aks")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("aks")
			}
			return err
		}
	}

	return nil
}

func (m *ExternalClusterMDPhase) validateState(formats strfmt.Registry) error {
	if swag.IsZero(m.State) { // not required
		return nil
	}

	if err := m.State.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("state")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("state")
		}
		return err
	}

	return nil
}

// ContextValidate validate this external cluster m d phase based on the context it is used
func (m *ExternalClusterMDPhase) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateAks(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateState(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ExternalClusterMDPhase) contextValidateAks(ctx context.Context, formats strfmt.Registry) error {

	if m.Aks != nil {
		if err := m.Aks.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("aks")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("aks")
			}
			return err
		}
	}

	return nil
}

func (m *ExternalClusterMDPhase) contextValidateState(ctx context.Context, formats strfmt.Registry) error {

	if err := m.State.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("state")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("state")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ExternalClusterMDPhase) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ExternalClusterMDPhase) UnmarshalBinary(b []byte) error {
	var res ExternalClusterMDPhase
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
