// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// DatacenterSpecOpenstack DatacenterSpecOpenstack describes an OpenStack datacenter
//
// swagger:model DatacenterSpecOpenstack
type DatacenterSpecOpenstack struct {

	// auth URL
	AuthURL string `json:"auth_url,omitempty"`

	// availability zone
	AvailabilityZone string `json:"availability_zone,omitempty"`

	// Used for automatic network creation
	DNSServers []string `json:"dns_servers"`

	// Optional
	EnforceFloatingIP bool `json:"enforce_floating_ip,omitempty"`

	// Optional
	IgnoreVolumeAZ bool `json:"ignore_volume_az,omitempty"`

	// Optional: Gets mapped to the "manage-security-groups" setting in the cloud config.
	// See https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#load-balancer
	// This setting defaults to true.
	ManageSecurityGroups bool `json:"manage_security_groups,omitempty"`

	// Optional: Gets mapped to the "use-octavia" setting in the cloud config.
	// This setting defaults to true.
	UseOctavia bool `json:"use_octavia,omitempty"`

	// region
	Region string `json:"region,omitempty"`

	// Optional: Gets mapped to the "trust-device-path" setting in the cloud config.
	// See https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#block-storage
	// This setting defaults to false.
	TrustDevicePath bool `json:"trust_device_path,omitempty"`

	// images
	Images ImageList `json:"images,omitempty"`

	// node size requirements
	NodeSizeRequirements *OpenstackNodeSizeRequirements `json:"node_size_requirements,omitempty"`
}

// Validate validates this datacenter spec openstack
func (m *DatacenterSpecOpenstack) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateImages(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNodeSizeRequirements(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *DatacenterSpecOpenstack) validateImages(formats strfmt.Registry) error {

	if swag.IsZero(m.Images) { // not required
		return nil
	}

	if err := m.Images.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("images")
		}
		return err
	}

	return nil
}

func (m *DatacenterSpecOpenstack) validateNodeSizeRequirements(formats strfmt.Registry) error {

	if swag.IsZero(m.NodeSizeRequirements) { // not required
		return nil
	}

	if m.NodeSizeRequirements != nil {
		if err := m.NodeSizeRequirements.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("node_size_requirements")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *DatacenterSpecOpenstack) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *DatacenterSpecOpenstack) UnmarshalBinary(b []byte) error {
	var res DatacenterSpecOpenstack
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
