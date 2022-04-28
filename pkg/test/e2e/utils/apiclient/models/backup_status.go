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

// BackupStatus backup status
//
// swagger:model BackupStatus
type BackupStatus struct {

	// backup message
	BackupMessage string `json:"backupMessage,omitempty"`

	// backup name
	BackupName string `json:"backupName,omitempty"`

	// delete job name
	DeleteJobName string `json:"deleteJobName,omitempty"`

	// delete message
	DeleteMessage string `json:"deleteMessage,omitempty"`

	// job name
	JobName string `json:"jobName,omitempty"`

	// backup finished time
	// Format: date-time
	BackupFinishedTime Time `json:"backupFinishedTime,omitempty"`

	// backup phase
	BackupPhase BackupStatusPhase `json:"backupPhase,omitempty"`

	// backup start time
	// Format: date-time
	BackupStartTime Time `json:"backupStartTime,omitempty"`

	// delete finished time
	// Format: date-time
	DeleteFinishedTime Time `json:"deleteFinishedTime,omitempty"`

	// delete phase
	DeletePhase BackupStatusPhase `json:"deletePhase,omitempty"`

	// delete start time
	// Format: date-time
	DeleteStartTime Time `json:"deleteStartTime,omitempty"`

	// scheduled time
	// Format: date-time
	ScheduledTime Time `json:"scheduledTime,omitempty"`
}

// Validate validates this backup status
func (m *BackupStatus) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateBackupFinishedTime(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateBackupPhase(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateBackupStartTime(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateDeleteFinishedTime(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateDeletePhase(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateDeleteStartTime(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateScheduledTime(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *BackupStatus) validateBackupFinishedTime(formats strfmt.Registry) error {
	if swag.IsZero(m.BackupFinishedTime) { // not required
		return nil
	}

	if err := m.BackupFinishedTime.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("backupFinishedTime")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("backupFinishedTime")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) validateBackupPhase(formats strfmt.Registry) error {
	if swag.IsZero(m.BackupPhase) { // not required
		return nil
	}

	if err := m.BackupPhase.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("backupPhase")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("backupPhase")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) validateBackupStartTime(formats strfmt.Registry) error {
	if swag.IsZero(m.BackupStartTime) { // not required
		return nil
	}

	if err := m.BackupStartTime.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("backupStartTime")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("backupStartTime")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) validateDeleteFinishedTime(formats strfmt.Registry) error {
	if swag.IsZero(m.DeleteFinishedTime) { // not required
		return nil
	}

	if err := m.DeleteFinishedTime.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("deleteFinishedTime")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("deleteFinishedTime")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) validateDeletePhase(formats strfmt.Registry) error {
	if swag.IsZero(m.DeletePhase) { // not required
		return nil
	}

	if err := m.DeletePhase.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("deletePhase")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("deletePhase")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) validateDeleteStartTime(formats strfmt.Registry) error {
	if swag.IsZero(m.DeleteStartTime) { // not required
		return nil
	}

	if err := m.DeleteStartTime.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("deleteStartTime")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("deleteStartTime")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) validateScheduledTime(formats strfmt.Registry) error {
	if swag.IsZero(m.ScheduledTime) { // not required
		return nil
	}

	if err := m.ScheduledTime.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("scheduledTime")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("scheduledTime")
		}
		return err
	}

	return nil
}

// ContextValidate validate this backup status based on the context it is used
func (m *BackupStatus) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateBackupFinishedTime(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateBackupPhase(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateBackupStartTime(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateDeleteFinishedTime(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateDeletePhase(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateDeleteStartTime(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateScheduledTime(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *BackupStatus) contextValidateBackupFinishedTime(ctx context.Context, formats strfmt.Registry) error {

	if err := m.BackupFinishedTime.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("backupFinishedTime")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("backupFinishedTime")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) contextValidateBackupPhase(ctx context.Context, formats strfmt.Registry) error {

	if err := m.BackupPhase.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("backupPhase")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("backupPhase")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) contextValidateBackupStartTime(ctx context.Context, formats strfmt.Registry) error {

	if err := m.BackupStartTime.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("backupStartTime")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("backupStartTime")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) contextValidateDeleteFinishedTime(ctx context.Context, formats strfmt.Registry) error {

	if err := m.DeleteFinishedTime.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("deleteFinishedTime")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("deleteFinishedTime")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) contextValidateDeletePhase(ctx context.Context, formats strfmt.Registry) error {

	if err := m.DeletePhase.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("deletePhase")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("deletePhase")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) contextValidateDeleteStartTime(ctx context.Context, formats strfmt.Registry) error {

	if err := m.DeleteStartTime.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("deleteStartTime")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("deleteStartTime")
		}
		return err
	}

	return nil
}

func (m *BackupStatus) contextValidateScheduledTime(ctx context.Context, formats strfmt.Registry) error {

	if err := m.ScheduledTime.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("scheduledTime")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("scheduledTime")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *BackupStatus) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *BackupStatus) UnmarshalBinary(b []byte) error {
	var res BackupStatus
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
