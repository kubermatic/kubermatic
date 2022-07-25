// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// VpcConfigRequest vpc config request
//
// swagger:model VpcConfigRequest
type VpcConfigRequest struct {

	// Specify one or more security groups for the cross-account elastic network
	// interfaces that Amazon EKS creates to use to allow communication between
	// your nodes and the Kubernetes control plane.
	// For more information, see Amazon EKS security group considerations (https://docs.aws.amazon.com/eks/latest/userguide/sec-group-reqs.html)
	// in the Amazon EKS User Guide .
	SecurityGroupIds []string `json:"securityGroupIds"`

	// Specify subnets for your Amazon EKS nodes. Amazon EKS creates cross-account
	// elastic network interfaces in these subnets to allow communication between
	// your nodes and the Kubernetes control plane.
	SubnetIds []string `json:"subnetIds"`

	// The VPC associated with your cluster.
	VpcID string `json:"vpcId,omitempty"`
}

// Validate validates this vpc config request
func (m *VpcConfigRequest) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this vpc config request based on context it is used
func (m *VpcConfigRequest) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *VpcConfigRequest) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *VpcConfigRequest) UnmarshalBinary(b []byte) error {
	var res VpcConfigRequest
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
