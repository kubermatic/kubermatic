// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
)

// ClockOffsetTimezone ClockOffsetTimezone sets the guest clock to the specified timezone.
//
// Zone name follows the TZ environment variable format (e.g. 'America/New_York').
//
// swagger:model ClockOffsetTimezone
type ClockOffsetTimezone string

// Validate validates this clock offset timezone
func (m ClockOffsetTimezone) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this clock offset timezone based on context it is used
func (m ClockOffsetTimezone) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
