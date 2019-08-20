// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"strconv"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
)

// AWSVPC AWSVPC represents a object of AWS VPC.
// swagger:model AWSVPC
type AWSVPC struct {

	// The primary IPv4 CIDR block for the VPC.
	CidrBlock string `json:"cidrBlock,omitempty"`

	// Information about the IPv4 CIDR blocks associated with the VPC.
	CidrBlockAssociationSet []*AWSVpcCidrBlockAssociation `json:"cidrBlockAssociationSet"`

	// The ID of the set of DHCP options you've associated with the VPC (or default
	// if the default options are associated with the VPC).
	DhcpOptionsID string `json:"dhcpOptionsId,omitempty"`

	// The allowed tenancy of instances launched into the VPC.
	InstanceTenancy string `json:"instanceTenancy,omitempty"`

	// Information about the IPv6 CIDR blocks associated with the VPC.
	IPV6CidrBlockAssociationSet []*AWSVpcIPV6CidrBlockAssociation `json:"ipv6CidrBlockAssociationSet"`

	// Indicates whether the VPC is the default VPC.
	IsDefault bool `json:"isDefault,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// The ID of the AWS account that owns the VPC.
	OwnerID string `json:"ownerId,omitempty"`

	// The current state of the VPC.
	State string `json:"state,omitempty"`

	// Any tags assigned to the VPC.
	Tags []*AWSTag `json:"tags"`

	// The ID of the VPC.
	VpcID string `json:"vpcId,omitempty"`
}

// Validate validates this a w s v p c
func (m *AWSVPC) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCidrBlockAssociationSet(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateIPV6CidrBlockAssociationSet(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateTags(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *AWSVPC) validateCidrBlockAssociationSet(formats strfmt.Registry) error {

	if swag.IsZero(m.CidrBlockAssociationSet) { // not required
		return nil
	}

	for i := 0; i < len(m.CidrBlockAssociationSet); i++ {
		if swag.IsZero(m.CidrBlockAssociationSet[i]) { // not required
			continue
		}

		if m.CidrBlockAssociationSet[i] != nil {
			if err := m.CidrBlockAssociationSet[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("cidrBlockAssociationSet" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *AWSVPC) validateIPV6CidrBlockAssociationSet(formats strfmt.Registry) error {

	if swag.IsZero(m.IPV6CidrBlockAssociationSet) { // not required
		return nil
	}

	for i := 0; i < len(m.IPV6CidrBlockAssociationSet); i++ {
		if swag.IsZero(m.IPV6CidrBlockAssociationSet[i]) { // not required
			continue
		}

		if m.IPV6CidrBlockAssociationSet[i] != nil {
			if err := m.IPV6CidrBlockAssociationSet[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("ipv6CidrBlockAssociationSet" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *AWSVPC) validateTags(formats strfmt.Registry) error {

	if swag.IsZero(m.Tags) { // not required
		return nil
	}

	for i := 0; i < len(m.Tags); i++ {
		if swag.IsZero(m.Tags[i]) { // not required
			continue
		}

		if m.Tags[i] != nil {
			if err := m.Tags[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("tags" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

// MarshalBinary interface implementation
func (m *AWSVPC) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AWSVPC) UnmarshalBinary(b []byte) error {
	var res AWSVPC
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
