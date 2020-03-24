// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// GlobalObjectKeySelector GlobalObjectKeySelector is needed as we can not use v1.SecretKeySelector
// because it is not cross namespace
//
// swagger:model GlobalObjectKeySelector
type GlobalObjectKeySelector struct {

	// API version of the referent.
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`

	// If referring to a piece of an object instead of an entire object, this string
	// should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2].
	// For example, if the object reference is to a container within a pod, this would take on a value like:
	// "spec.containers{name}" (where "name" refers to the name of the container that triggered
	// the event) or if no container name is specified "spec.containers[2]" (container with
	// index 2 in this pod). This syntax is chosen only to have some well-defined way of
	// referencing a part of an object.
	// TODO: this design is not final and this field is subject to change in the future.
	// +optional
	FieldPath string `json:"fieldPath,omitempty"`

	// key
	Key string `json:"key,omitempty"`

	// Kind of the referent.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	// +optional
	Kind string `json:"kind,omitempty"`

	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Specific resourceVersion to which this reference is made, if any.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency
	// +optional
	ResourceVersion string `json:"resourceVersion,omitempty"`

	// uid
	UID UID `json:"uid,omitempty"`
}

// Validate validates this global object key selector
func (m *GlobalObjectKeySelector) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateUID(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *GlobalObjectKeySelector) validateUID(formats strfmt.Registry) error {

	if swag.IsZero(m.UID) { // not required
		return nil
	}

	if err := m.UID.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("uid")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *GlobalObjectKeySelector) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *GlobalObjectKeySelector) UnmarshalBinary(b []byte) error {
	var res GlobalObjectKeySelector
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
