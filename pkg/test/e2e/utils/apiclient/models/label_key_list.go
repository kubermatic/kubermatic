// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
)

// LabelKeyList label key list
//
// swagger:model LabelKeyList
type LabelKeyList []string

// Validate validates this label key list
func (m LabelKeyList) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this label key list based on context it is used
func (m LabelKeyList) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
