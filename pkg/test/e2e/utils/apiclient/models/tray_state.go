// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
)

// TrayState TrayState indicates if a tray of a cdrom or floppy is open or closed.
//
// swagger:model TrayState
type TrayState string

// Validate validates this tray state
func (m TrayState) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this tray state based on context it is used
func (m TrayState) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
