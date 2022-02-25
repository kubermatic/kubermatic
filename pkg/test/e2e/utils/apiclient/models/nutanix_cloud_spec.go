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

// NutanixCloudSpec NutanixCloudSpec specifies the access data to Nutanix.
//
// swagger:model NutanixCloudSpec
type NutanixCloudSpec struct {

	// ClusterName is the Nutanix cluster that this user cluster will be deployed to.
	ClusterName string `json:"clusterName,omitempty"`

	// password
	Password string `json:"password,omitempty"`

	// ProjectName is the project that this cluster is deployed into. If none is given, no project will be used.
	// +optional
	ProjectName string `json:"projectName,omitempty"`

	// proxy URL
	ProxyURL string `json:"proxyURL,omitempty"`

	// username
	Username string `json:"username,omitempty"`

	// credentials reference
	CredentialsReference *GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// csi
	Csi *NutanixCSIConfig `json:"csi,omitempty"`
}

// Validate validates this nutanix cloud spec
func (m *NutanixCloudSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCredentialsReference(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateCsi(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *NutanixCloudSpec) validateCredentialsReference(formats strfmt.Registry) error {
	if swag.IsZero(m.CredentialsReference) { // not required
		return nil
	}

	if m.CredentialsReference != nil {
		if err := m.CredentialsReference.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("credentialsReference")
			}
			return err
		}
	}

	return nil
}

func (m *NutanixCloudSpec) validateCsi(formats strfmt.Registry) error {
	if swag.IsZero(m.Csi) { // not required
		return nil
	}

	if m.Csi != nil {
		if err := m.Csi.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("csi")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this nutanix cloud spec based on the context it is used
func (m *NutanixCloudSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateCredentialsReference(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateCsi(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *NutanixCloudSpec) contextValidateCredentialsReference(ctx context.Context, formats strfmt.Registry) error {

	if m.CredentialsReference != nil {
		if err := m.CredentialsReference.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("credentialsReference")
			}
			return err
		}
	}

	return nil
}

func (m *NutanixCloudSpec) contextValidateCsi(ctx context.Context, formats strfmt.Registry) error {

	if m.Csi != nil {
		if err := m.Csi.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("csi")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *NutanixCloudSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *NutanixCloudSpec) UnmarshalBinary(b []byte) error {
	var res NutanixCloudSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
