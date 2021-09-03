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

// BcBody bc body
//
// swagger:model bcBody
type BcBody struct {

	// backup credentials
	BackupCredentials *BackupCredentials `json:"backup_credentials,omitempty"`
}

// Validate validates this bc body
func (m *BcBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateBackupCredentials(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *BcBody) validateBackupCredentials(formats strfmt.Registry) error {
	if swag.IsZero(m.BackupCredentials) { // not required
		return nil
	}

	if m.BackupCredentials != nil {
		if err := m.BackupCredentials.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("backup_credentials")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this bc body based on the context it is used
func (m *BcBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateBackupCredentials(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *BcBody) contextValidateBackupCredentials(ctx context.Context, formats strfmt.Registry) error {

	if m.BackupCredentials != nil {
		if err := m.BackupCredentials.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("backup_credentials")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *BcBody) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *BcBody) UnmarshalBinary(b []byte) error {
	var res BcBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
