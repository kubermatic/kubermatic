// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	strfmt "github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// CreateClusterSpec CreateClusterSpec is the structure that is used to create cluster with its initial node deployment
// swagger:model CreateClusterSpec
type CreateClusterSpec struct {

	// cluster
	Cluster *Cluster `json:"cluster,omitempty"`

	// node deployment
	NodeDeployment *NodeDeployment `json:"nodeDeployment,omitempty"`
}

// Validate validates this create cluster spec
func (m *CreateClusterSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCluster(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNodeDeployment(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *CreateClusterSpec) validateCluster(formats strfmt.Registry) error {

	if swag.IsZero(m.Cluster) { // not required
		return nil
	}

	if m.Cluster != nil {
		if err := m.Cluster.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("cluster")
			}
			return err
		}
	}

	return nil
}

func (m *CreateClusterSpec) validateNodeDeployment(formats strfmt.Registry) error {

	if swag.IsZero(m.NodeDeployment) { // not required
		return nil
	}

	if m.NodeDeployment != nil {
		if err := m.NodeDeployment.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("nodeDeployment")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *CreateClusterSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *CreateClusterSpec) UnmarshalBinary(b []byte) error {
	var res CreateClusterSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
