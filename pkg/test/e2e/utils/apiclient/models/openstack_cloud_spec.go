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

// OpenstackCloudSpec OpenstackCloudSpec specifies access data to an OpenStack cloud.
//
// swagger:model OpenstackCloudSpec
type OpenstackCloudSpec struct {

	// application credential ID
	ApplicationCredentialID string `json:"applicationCredentialID,omitempty"`

	// application credential secret
	ApplicationCredentialSecret string `json:"applicationCredentialSecret,omitempty"`

	// domain
	Domain string `json:"domain,omitempty"`

	// FloatingIPPool holds the name of the public network
	// The public network is reachable from the outside world
	// and should provide the pool of IP addresses to choose from.
	//
	// When specified, all worker nodes will receive a public ip from this floating ip pool
	//
	// Note that the network is external if the "External" field is set to true
	FloatingIPPool string `json:"floatingIPPool,omitempty"`

	// IPv6SubnetID holds the ID of the subnet used for IPv6 networking.
	// If not provided, a new subnet will be created if IPv6 is enabled.
	// +optional
	IPV6SubnetID string `json:"ipv6SubnetID,omitempty"`

	// IPv6SubnetPool holds the name of the subnet pool used for creating new IPv6 subnets.
	// If not provided, the default IPv6 subnet pool will be used.
	// +optional
	IPV6SubnetPool string `json:"ipv6SubnetPool,omitempty"`

	// Network holds the name of the internal network
	// When specified, all worker nodes will be attached to this network. If not specified, a network, subnet & router will be created
	//
	// Note that the network is internal if the "External" field is set to false
	Network string `json:"network,omitempty"`

	// A CIDR range that will be used to allow access to the node port range in the security group to. Only applies if
	// the security group is generated by KKP and not preexisting.
	// If NodePortsAllowedIPRange nor NodePortsAllowedIPRanges is set, the node port range can be accessed from anywhere.
	NodePortsAllowedIPRange string `json:"nodePortsAllowedIPRange,omitempty"`

	// password
	Password string `json:"password,omitempty"`

	// project, formally known as tenant.
	Project string `json:"project,omitempty"`

	// project id, formally known as tenantID.
	ProjectID string `json:"projectID,omitempty"`

	// router ID
	RouterID string `json:"routerID,omitempty"`

	// security groups
	SecurityGroups string `json:"securityGroups,omitempty"`

	// subnet ID
	SubnetID string `json:"subnetID,omitempty"`

	// Used internally during cluster creation
	Token string `json:"token,omitempty"`

	// Whether or not to use Octavia for LoadBalancer type of Service
	// implementation instead of using Neutron-LBaaS.
	// Attention:Openstack CCM use Octavia as default load balancer
	// implementation since v1.17.0
	//
	// Takes precedence over the 'use_octavia' flag provided at datacenter
	// level if both are specified.
	// +optional
	UseOctavia bool `json:"useOctavia,omitempty"`

	// use token
	UseToken bool `json:"useToken,omitempty"`

	// username
	Username string `json:"username,omitempty"`

	// credentials reference
	CredentialsReference *GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// node ports allowed IP ranges
	NodePortsAllowedIPRanges *NetworkRanges `json:"nodePortsAllowedIPRanges,omitempty"`
}

// Validate validates this openstack cloud spec
func (m *OpenstackCloudSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCredentialsReference(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNodePortsAllowedIPRanges(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *OpenstackCloudSpec) validateCredentialsReference(formats strfmt.Registry) error {
	if swag.IsZero(m.CredentialsReference) { // not required
		return nil
	}

	if m.CredentialsReference != nil {
		if err := m.CredentialsReference.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("credentialsReference")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("credentialsReference")
			}
			return err
		}
	}

	return nil
}

func (m *OpenstackCloudSpec) validateNodePortsAllowedIPRanges(formats strfmt.Registry) error {
	if swag.IsZero(m.NodePortsAllowedIPRanges) { // not required
		return nil
	}

	if m.NodePortsAllowedIPRanges != nil {
		if err := m.NodePortsAllowedIPRanges.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("nodePortsAllowedIPRanges")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("nodePortsAllowedIPRanges")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this openstack cloud spec based on the context it is used
func (m *OpenstackCloudSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateCredentialsReference(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateNodePortsAllowedIPRanges(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *OpenstackCloudSpec) contextValidateCredentialsReference(ctx context.Context, formats strfmt.Registry) error {

	if m.CredentialsReference != nil {
		if err := m.CredentialsReference.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("credentialsReference")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("credentialsReference")
			}
			return err
		}
	}

	return nil
}

func (m *OpenstackCloudSpec) contextValidateNodePortsAllowedIPRanges(ctx context.Context, formats strfmt.Registry) error {

	if m.NodePortsAllowedIPRanges != nil {
		if err := m.NodePortsAllowedIPRanges.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("nodePortsAllowedIPRanges")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("nodePortsAllowedIPRanges")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *OpenstackCloudSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *OpenstackCloudSpec) UnmarshalBinary(b []byte) error {
	var res OpenstackCloudSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
