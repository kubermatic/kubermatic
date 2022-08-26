// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// AKS a k s
//
// swagger:model AKS
type AKS struct {

	// client ID
	ClientID string `json:"clientID,omitempty"`

	// client secret
	ClientSecret string `json:"clientSecret,omitempty"`

	// If datacenter is set, this preset is only applicable to the
	// configured datacenter.
	Datacenter string `json:"datacenter,omitempty"`

	// Only enabled presets will be available in the KKP dashboard.
	Enabled bool `json:"enabled,omitempty"`

	// subscription ID
	SubscriptionID string `json:"subscriptionID,omitempty"`

	// tenant ID
	TenantID string `json:"tenantID,omitempty"`
}

// Validate validates this a k s
func (m *AKS) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this a k s based on context it is used
func (m *AKS) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AKS) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AKS) UnmarshalBinary(b []byte) error {
	var res AKS
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
