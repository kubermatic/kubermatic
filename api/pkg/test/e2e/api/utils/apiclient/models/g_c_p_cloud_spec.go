// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// GCPCloudSpec GCPCloudSpec specifies access data to GCP.
//
// swagger:model GCPCloudSpec
type GCPCloudSpec struct {

	// network
	Network string `json:"network,omitempty"`

	// service account
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// subnetwork
	Subnetwork string `json:"subnetwork,omitempty"`

	// credentials reference
	CredentialsReference GlobalSecretKeySelector `json:"credentialsReference,omitempty"`
}

// Validate validates this g c p cloud spec
func (m *GCPCloudSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCredentialsReference(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *GCPCloudSpec) validateCredentialsReference(formats strfmt.Registry) error {

	if swag.IsZero(m.CredentialsReference) { // not required
		return nil
	}

	if err := m.CredentialsReference.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("credentialsReference")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *GCPCloudSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *GCPCloudSpec) UnmarshalBinary(b []byte) error {
	var res GCPCloudSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
