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

// EKSClusterSpec e k s cluster spec
//
// swagger:model EKSClusterSpec
type EKSClusterSpec struct {

	// The Amazon Resource Name (ARN) of the IAM role that provides permissions
	// for the Kubernetes control plane to make calls to AWS API operations on your
	// behalf. For more information, see Amazon EKS Service IAM Role (https://docs.aws.amazon.com/eks/latest/userguide/service_IAM_role.html)
	// in the Amazon EKS User Guide .
	//
	// RoleArn is a required field
	RoleArn string `json:"roleArn,omitempty"`

	// The desired Kubernetes version for your cluster. If you don't specify a value
	// here, the latest version available in Amazon EKS is used.
	Version string `json:"version,omitempty"`

	// vpc config request
	VpcConfigRequest *VpcConfigRequest `json:"vpcConfigRequest,omitempty"`
}

// Validate validates this e k s cluster spec
func (m *EKSClusterSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateVpcConfigRequest(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *EKSClusterSpec) validateVpcConfigRequest(formats strfmt.Registry) error {
	if swag.IsZero(m.VpcConfigRequest) { // not required
		return nil
	}

	if m.VpcConfigRequest != nil {
		if err := m.VpcConfigRequest.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vpcConfigRequest")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this e k s cluster spec based on the context it is used
func (m *EKSClusterSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateVpcConfigRequest(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *EKSClusterSpec) contextValidateVpcConfigRequest(ctx context.Context, formats strfmt.Registry) error {

	if m.VpcConfigRequest != nil {
		if err := m.VpcConfigRequest.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vpcConfigRequest")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *EKSClusterSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *EKSClusterSpec) UnmarshalBinary(b []byte) error {
	var res EKSClusterSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
