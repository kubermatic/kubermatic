// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
)

// AuditPolicyPreset AuditPolicyPreset refers to a pre-defined set of audit policy rules. Supported values
// are `metadata`, `recommended` and `minimal`. See KKP documentation for what each policy preset includes.
//
// swagger:model AuditPolicyPreset
type AuditPolicyPreset string

// Validate validates this audit policy preset
func (m AuditPolicyPreset) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this audit policy preset based on context it is used
func (m AuditPolicyPreset) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
