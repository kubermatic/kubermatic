// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// Openstack openstack
//
// swagger:model Openstack
type Openstack struct {

	// application credential ID
	ApplicationCredentialID string `json:"applicationCredentialID,omitempty"`

	// application credential secret
	ApplicationCredentialSecret string `json:"applicationCredentialSecret,omitempty"`

	// If datacenter is set, this preset is only applicable to the
	// configured datacenter.
	Datacenter string `json:"datacenter,omitempty"`

	// domain
	Domain string `json:"domain,omitempty"`

	// Only enabled presets will be available in the KKP dashboard.
	Enabled bool `json:"enabled,omitempty"`

	// floating IP pool
	FloatingIPPool string `json:"floatingIPPool,omitempty"`

	// network
	Network string `json:"network,omitempty"`

	// password
	Password string `json:"password,omitempty"`

	// project
	Project string `json:"project,omitempty"`

	// project ID
	ProjectID string `json:"projectID,omitempty"`

	// router ID
	RouterID string `json:"routerID,omitempty"`

	// security groups
	SecurityGroups string `json:"securityGroups,omitempty"`

	// subnet ID
	SubnetID string `json:"subnetID,omitempty"`

	// use token
	UseToken bool `json:"useToken,omitempty"`

	// username
	Username string `json:"username,omitempty"`
}

// Validate validates this openstack
func (m *Openstack) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this openstack based on context it is used
func (m *Openstack) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *Openstack) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Openstack) UnmarshalBinary(b []byte) error {
	var res Openstack
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
