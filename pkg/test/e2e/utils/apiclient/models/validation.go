// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// Validation validation
//
// swagger:model Validation
type Validation struct {

	// open API v3 schema
	OpenAPIV3Schema *JSONSchemaProps `json:"openAPIV3Schema,omitempty"`
}

// Validate validates this validation
func (m *Validation) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateOpenAPIV3Schema(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Validation) validateOpenAPIV3Schema(formats strfmt.Registry) error {
	if swag.IsZero(m.OpenAPIV3Schema) { // not required
		return nil
	}

	if m.OpenAPIV3Schema != nil {
		if err := m.OpenAPIV3Schema.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("openAPIV3Schema")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this validation based on the context it is used
func (m *Validation) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateOpenAPIV3Schema(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Validation) contextValidateOpenAPIV3Schema(ctx context.Context, formats strfmt.Registry) error {

	if m.OpenAPIV3Schema != nil {
		if err := m.OpenAPIV3Schema.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("openAPIV3Schema")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *Validation) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Validation) UnmarshalBinary(b []byte) error {
	var res Validation
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
