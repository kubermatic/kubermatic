// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// ServiceAccountToken ServiceAccountToken represent an API service account token
//
// swagger:model ServiceAccountToken
type ServiceAccountToken struct {

	// Annotations that can be added to the resource
	Annotations map[string]string `json:"annotations,omitempty"`

	// CreationTimestamp is a timestamp representing the server time when this object was created.
	// Format: date-time
	CreationTimestamp strfmt.DateTime `json:"creationTimestamp,omitempty"`

	// DeletionTimestamp is a timestamp representing the server time when this object was deleted.
	// Format: date-time
	DeletionTimestamp strfmt.DateTime `json:"deletionTimestamp,omitempty"`

	// Expiry is a timestamp representing the time when this token will expire.
	// Format: date-time
	Expiry strfmt.DateTime `json:"expiry,omitempty"`

	// ID unique value that identifies the resource generated by the server. Read-Only.
	ID string `json:"id,omitempty"`

	// Invalidated indicates if the token must be regenerated
	Invalidated bool `json:"invalidated,omitempty"`

	// Name represents human readable name for the resource
	Name string `json:"name,omitempty"`

	// Token the JWT token
	Token string `json:"token,omitempty"`
}

// Validate validates this service account token
func (m *ServiceAccountToken) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCreationTimestamp(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateDeletionTimestamp(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateExpiry(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ServiceAccountToken) validateCreationTimestamp(formats strfmt.Registry) error {
	if swag.IsZero(m.CreationTimestamp) { // not required
		return nil
	}

	if err := validate.FormatOf("creationTimestamp", "body", "date-time", m.CreationTimestamp.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *ServiceAccountToken) validateDeletionTimestamp(formats strfmt.Registry) error {
	if swag.IsZero(m.DeletionTimestamp) { // not required
		return nil
	}

	if err := validate.FormatOf("deletionTimestamp", "body", "date-time", m.DeletionTimestamp.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *ServiceAccountToken) validateExpiry(formats strfmt.Registry) error {
	if swag.IsZero(m.Expiry) { // not required
		return nil
	}

	if err := validate.FormatOf("expiry", "body", "date-time", m.Expiry.String(), formats); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this service account token based on context it is used
func (m *ServiceAccountToken) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *ServiceAccountToken) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ServiceAccountToken) UnmarshalBinary(b []byte) error {
	var res ServiceAccountToken
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
