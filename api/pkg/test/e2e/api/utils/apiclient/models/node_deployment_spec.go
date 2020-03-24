// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// NodeDeploymentSpec NodeDeploymentSpec node deployment specification
//
// swagger:model NodeDeploymentSpec
type NodeDeploymentSpec struct {

	// dynamic config
	DynamicConfig bool `json:"dynamicConfig,omitempty"`

	// paused
	Paused bool `json:"paused,omitempty"`

	// replicas
	// Required: true
	Replicas *int32 `json:"replicas"`

	// template
	// Required: true
	Template *NodeSpec `json:"template"`
}

// Validate validates this node deployment spec
func (m *NodeDeploymentSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateReplicas(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateTemplate(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *NodeDeploymentSpec) validateReplicas(formats strfmt.Registry) error {

	if err := validate.Required("replicas", "body", m.Replicas); err != nil {
		return err
	}

	return nil
}

func (m *NodeDeploymentSpec) validateTemplate(formats strfmt.Registry) error {

	if err := validate.Required("template", "body", m.Template); err != nil {
		return err
	}

	if m.Template != nil {
		if err := m.Template.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("template")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *NodeDeploymentSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *NodeDeploymentSpec) UnmarshalBinary(b []byte) error {
	var res NodeDeploymentSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
