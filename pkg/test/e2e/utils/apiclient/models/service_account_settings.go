// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// ServiceAccountSettings service account settings
//
// swagger:model ServiceAccountSettings
type ServiceAccountSettings struct {

	// APIAudiences are the Identifiers of the API
	// If this is not specified, it will be set to a single element list containing the issuer URL
	APIAudiences []string `json:"apiAudiences"`

	// Issuer is the identifier of the service account token issuer
	// If this is not specified, it will be set to the URL of apiserver by default
	Issuer string `json:"issuer,omitempty"`

	// token volume projection enabled
	TokenVolumeProjectionEnabled bool `json:"tokenVolumeProjectionEnabled,omitempty"`
}

// Validate validates this service account settings
func (m *ServiceAccountSettings) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this service account settings based on context it is used
func (m *ServiceAccountSettings) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *ServiceAccountSettings) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ServiceAccountSettings) UnmarshalBinary(b []byte) error {
	var res ServiceAccountSettings
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
