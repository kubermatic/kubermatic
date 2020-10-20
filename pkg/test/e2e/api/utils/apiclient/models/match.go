// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// Match Match contains the constraint to resource matching data
//
// swagger:model Match
type Match struct {

	// ExcludedNamespaces is a list of namespace names. If defined, a constraint will only apply to resources not in a listed namespace.
	ExcludedNamespaces []string `json:"excludedNamespaces"`

	// Kinds accepts a list of objects with apiGroups and kinds fields that list the groups/kinds of objects to which
	// the constraint will apply. If multiple groups/kinds objects are specified, only one match is needed for the resource to be in scope
	Kinds []*Kind `json:"kinds"`

	// Namespaces is a list of namespace names. If defined, a constraint will only apply to resources in a listed namespace.
	Namespaces []string `json:"namespaces"`

	// Scope accepts *, Cluster, or Namespaced which determines if cluster-scoped and/or namesapced-scoped resources are selected. (defaults to *)
	Scope string `json:"scope,omitempty"`

	// label selector
	LabelSelector *LabelSelector `json:"labelSelector,omitempty"`

	// namespace selector
	NamespaceSelector *LabelSelector `json:"namespaceSelector,omitempty"`
}

// Validate validates this match
func (m *Match) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateKinds(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateLabelSelector(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNamespaceSelector(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Match) validateKinds(formats strfmt.Registry) error {

	if swag.IsZero(m.Kinds) { // not required
		return nil
	}

	for i := 0; i < len(m.Kinds); i++ {
		if swag.IsZero(m.Kinds[i]) { // not required
			continue
		}

		if m.Kinds[i] != nil {
			if err := m.Kinds[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("kinds" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *Match) validateLabelSelector(formats strfmt.Registry) error {

	if swag.IsZero(m.LabelSelector) { // not required
		return nil
	}

	if m.LabelSelector != nil {
		if err := m.LabelSelector.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("labelSelector")
			}
			return err
		}
	}

	return nil
}

func (m *Match) validateNamespaceSelector(formats strfmt.Registry) error {

	if swag.IsZero(m.NamespaceSelector) { // not required
		return nil
	}

	if m.NamespaceSelector != nil {
		if err := m.NamespaceSelector.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("namespaceSelector")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *Match) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Match) UnmarshalBinary(b []byte) error {
	var res Match
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
