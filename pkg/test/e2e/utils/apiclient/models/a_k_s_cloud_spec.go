// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// AKSCloudSpec a k s cloud spec
//
// swagger:model AKSCloudSpec
type AKSCloudSpec struct {

	// client ID
	ClientID string `json:"clientID,omitempty"`

	// client secret
	ClientSecret string `json:"clientSecret,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// resource group
	ResourceGroup string `json:"resourceGroup,omitempty"`

	// subscription ID
	SubscriptionID string `json:"subscriptionID,omitempty"`

	// tenant ID
	TenantID string `json:"tenantID,omitempty"`
}

// Validate validates this a k s cloud spec
func (m *AKSCloudSpec) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this a k s cloud spec based on context it is used
func (m *AKSCloudSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AKSCloudSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AKSCloudSpec) UnmarshalBinary(b []byte) error {
	var res AKSCloudSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
