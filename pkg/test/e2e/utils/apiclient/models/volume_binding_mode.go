// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
)

// VolumeBindingMode VolumeBindingMode indicates how PersistentVolumeClaims should be bound.
//
// +enum
//
// swagger:model VolumeBindingMode
type VolumeBindingMode string

// Validate validates this volume binding mode
func (m VolumeBindingMode) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this volume binding mode based on context it is used
func (m VolumeBindingMode) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
