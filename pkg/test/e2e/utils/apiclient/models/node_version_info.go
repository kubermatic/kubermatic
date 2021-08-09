// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// NodeVersionInfo NodeVersionInfo node version information
//
// swagger:model NodeVersionInfo
type NodeVersionInfo struct {

	// kubelet
	Kubelet string `json:"kubelet,omitempty"`
}

// Validate validates this node version info
func (m *NodeVersionInfo) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this node version info based on context it is used
func (m *NodeVersionInfo) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *NodeVersionInfo) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *NodeVersionInfo) UnmarshalBinary(b []byte) error {
	var res NodeVersionInfo
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
