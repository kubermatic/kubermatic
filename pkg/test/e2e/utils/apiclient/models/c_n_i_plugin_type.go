// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
)

// CNIPluginType CNIPluginType defines the type of CNI plugin installed.
//
// Possible values are `canal`, `cilium` or `none`.
//
// swagger:model CNIPluginType
type CNIPluginType string

// Validate validates this c n i plugin type
func (m CNIPluginType) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this c n i plugin type based on context it is used
func (m CNIPluginType) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
