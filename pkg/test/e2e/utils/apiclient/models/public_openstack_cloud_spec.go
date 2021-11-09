// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// PublicOpenstackCloudSpec PublicOpenstackCloudSpec is a public counterpart of apiv1.OpenstackCloudSpec.
//
// swagger:model PublicOpenstackCloudSpec
type PublicOpenstackCloudSpec struct {

	// domain
	Domain string `json:"domain,omitempty"`

	// floating IP pool
	FloatingIPPool string `json:"floatingIpPool,omitempty"`

	// network
	Network string `json:"network,omitempty"`

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

	// tenant
	Tenant string `json:"tenant,omitempty"`

	// tenant ID
	TenantID string `json:"tenantID,omitempty"`
}

// Validate validates this public openstack cloud spec
func (m *PublicOpenstackCloudSpec) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this public openstack cloud spec based on context it is used
func (m *PublicOpenstackCloudSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *PublicOpenstackCloudSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *PublicOpenstackCloudSpec) UnmarshalBinary(b []byte) error {
	var res PublicOpenstackCloudSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
