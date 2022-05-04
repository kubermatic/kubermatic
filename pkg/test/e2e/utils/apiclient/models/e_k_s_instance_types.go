// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
)

// EKSInstanceTypes EKSInstanceTypes represents a list of EKS Instance Types for node group.
//
// swagger:model EKSInstanceTypes
type EKSInstanceTypes []string

// Validate validates this e k s instance types
func (m EKSInstanceTypes) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this e k s instance types based on context it is used
func (m EKSInstanceTypes) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
