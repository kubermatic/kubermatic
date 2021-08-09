// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// OIDCSpec OIDCSpec contains OIDC params that can be used to access user cluster.
//
// swagger:model OIDCSpec
type OIDCSpec struct {

	// client ID
	ClientID string `json:"clientId,omitempty"`

	// client secret
	ClientSecret string `json:"clientSecret,omitempty"`

	// issuer URL
	IssuerURL string `json:"issuerUrl,omitempty"`
}

// Validate validates this o ID c spec
func (m *OIDCSpec) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this o ID c spec based on context it is used
func (m *OIDCSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *OIDCSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *OIDCSpec) UnmarshalBinary(b []byte) error {
	var res OIDCSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
