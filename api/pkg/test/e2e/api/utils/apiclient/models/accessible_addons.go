// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"
)

// AccessibleAddons AccessibleAddons represents an array of addons that can be configured in the user clusters.
// swagger:model AccessibleAddons
type AccessibleAddons []string

// Validate validates this accessible addons
func (m AccessibleAddons) Validate(formats strfmt.Registry) error {
	return nil
}
