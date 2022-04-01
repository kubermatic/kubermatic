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

// EtcdBackupRestore EtcdBackupRestore holds the configuration of the automatic backup and restores.
//
// swagger:model EtcdBackupRestore
type EtcdBackupRestore struct {

	// DefaultDestination marks the default destination that will be used for the default etcd backup config which is
	// created for every user cluster. Has to correspond to a destination in Destinations.
	// If removed, it removes the related default etcd backup configs.
	DefaultDestination string `json:"defaultDestination,omitempty"`

	// Destinations stores all the possible destinations where the backups for the Seed can be stored. If not empty,
	// it enables automatic backup and restore for the seed.
	Destinations map[string]BackupDestination `json:"destinations,omitempty"`
}

// Validate validates this etcd backup restore
func (m *EtcdBackupRestore) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateDestinations(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *EtcdBackupRestore) validateDestinations(formats strfmt.Registry) error {
	if swag.IsZero(m.Destinations) { // not required
		return nil
	}

	for k := range m.Destinations {

		if err := validate.Required("destinations"+"."+k, "body", m.Destinations[k]); err != nil {
			return err
		}
		if val, ok := m.Destinations[k]; ok {
			if err := val.Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("destinations" + "." + k)
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("destinations" + "." + k)
				}
				return err
			}
		}

	}

	return nil
}

// ContextValidate validate this etcd backup restore based on the context it is used
func (m *EtcdBackupRestore) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateDestinations(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *EtcdBackupRestore) contextValidateDestinations(ctx context.Context, formats strfmt.Registry) error {

	for k := range m.Destinations {

		if val, ok := m.Destinations[k]; ok {
			if err := val.ContextValidate(ctx, formats); err != nil {
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *EtcdBackupRestore) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *EtcdBackupRestore) UnmarshalBinary(b []byte) error {
	var res EtcdBackupRestore
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
