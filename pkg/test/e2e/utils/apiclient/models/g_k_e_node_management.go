// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// GKENodeManagement GKENodeManagement defines the set of node management
// services turned on for the node pool.
//
// swagger:model GKENodeManagement
type GKENodeManagement struct {

	// AutoRepair: A flag that specifies whether the node auto-repair is
	// enabled for the node pool. If enabled, the nodes in this node pool
	// will be monitored and, if they fail health checks too many times, an
	// automatic repair action will be triggered.
	AutoRepair bool `json:"autoRepair,omitempty"`

	// AutoUpgrade: A flag that specifies whether node auto-upgrade is
	// enabled for the node pool. If enabled, node auto-upgrade helps keep
	// the nodes in your node pool up to date with the latest release
	// version of Kubernetes.
	AutoUpgrade bool `json:"autoUpgrade,omitempty"`
}

// Validate validates this g k e node management
func (m *GKENodeManagement) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this g k e node management based on context it is used
func (m *GKENodeManagement) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *GKENodeManagement) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *GKENodeManagement) UnmarshalBinary(b []byte) error {
	var res GKENodeManagement
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
