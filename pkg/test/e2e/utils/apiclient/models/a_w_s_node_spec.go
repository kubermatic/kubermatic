// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// AWSNodeSpec AWSNodeSpec aws specific node settings
//
// swagger:model AWSNodeSpec
type AWSNodeSpec struct {

	// ami to use. Will be defaulted to a ami for your selected operating system and region. Only set this when you know what you do.
	AMI string `json:"ami,omitempty"`

	// This flag controls a property of the AWS instance. When set the AWS instance will get a public IP address
	// assigned during launch overriding a possible setting in the used AWS subnet.
	AssignPublicIP bool `json:"assignPublicIP,omitempty"`

	// Availability zone in which to place the node. It is coupled with the subnet to which the node will belong.
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// instance type
	// Example: t2.micro
	// Required: true
	InstanceType *string `json:"instanceType"`

	// IsSpotInstance indicates whether the created machine is an aws ec2 spot instance or on-demand ec2 instance.
	IsSpotInstance bool `json:"isSpotInstance,omitempty"`

	// SpotInstanceInterruptionBehavior sets the interruption behavior for the spot instance when capacity is no longer
	// available at the price you specified, if there is no capacity, or if a constraint cannot be met. Charges for EBS
	// volume storage apply when an instance is stopped.
	SpotInstanceInterruptionBehavior string `json:"spotInstanceInterruptionBehavior,omitempty"`

	// SpotInstanceMaxPrice is the maximum price you are willing to pay per instance hour. Your instance runs when
	// your maximum price is greater than the Spot Price.
	SpotInstanceMaxPrice string `json:"spotInstanceMaxPrice,omitempty"`

	// SpotInstancePersistentRequest ensures that your request will be submitted every time your Spot Instance is terminated.
	SpotInstancePersistentRequest bool `json:"spotInstancePersistentRequest,omitempty"`

	// The VPC subnet to which the node shall be connected.
	SubnetID string `json:"subnetID,omitempty"`

	// additional instance tags
	Tags map[string]string `json:"tags,omitempty"`

	// size of the volume in gb. Only one volume will be created
	// Required: true
	VolumeSize *int64 `json:"diskSize"`

	// volume type
	// Example: gp2, io1, st1, sc1, standard
	// Required: true
	VolumeType *string `json:"volumeType"`
}

// Validate validates this a w s node spec
func (m *AWSNodeSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateInstanceType(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVolumeSize(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVolumeType(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *AWSNodeSpec) validateInstanceType(formats strfmt.Registry) error {

	if err := validate.Required("instanceType", "body", m.InstanceType); err != nil {
		return err
	}

	return nil
}

func (m *AWSNodeSpec) validateVolumeSize(formats strfmt.Registry) error {

	if err := validate.Required("diskSize", "body", m.VolumeSize); err != nil {
		return err
	}

	return nil
}

func (m *AWSNodeSpec) validateVolumeType(formats strfmt.Registry) error {

	if err := validate.Required("volumeType", "body", m.VolumeType); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this a w s node spec based on context it is used
func (m *AWSNodeSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AWSNodeSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AWSNodeSpec) UnmarshalBinary(b []byte) error {
	var res AWSNodeSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
