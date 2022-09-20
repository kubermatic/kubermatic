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

// EKSMachineDeploymentCloudSpec e k s machine deployment cloud spec
//
// swagger:model EKSMachineDeploymentCloudSpec
type EKSMachineDeploymentCloudSpec struct {

	// The AMI type for your node group. GPU instance types should use the AL2_x86_64_GPU
	// AMI type. Non-GPU instances should use the AL2_x86_64 AMI type. Arm instances
	// should use the AL2_ARM_64 AMI type. All types use the Amazon EKS optimized
	// Amazon Linux 2 AMI. If you specify launchTemplate, and your launch template
	// uses a custom AMI, then don't specify amiType, or the node group deployment
	// will fail. For more information about using launch templates with Amazon
	// EKS, see Launch template support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	AmiType string `json:"amiType,omitempty"`

	// The architecture of the machine image.
	Architecture string `json:"architecture,omitempty"`

	// The capacity type for your node group. Possible values ON_DEMAND | SPOT
	CapacityType string `json:"capacityType,omitempty"`

	// The Unix epoch timestamp in seconds for when the managed node group was created.
	// Format: date-time
	CreatedAt strfmt.DateTime `json:"createdAt,omitempty"`

	// The root device disk size (in GiB) for your node group instances. The default
	// disk size is 20 GiB. If you specify launchTemplate, then don't specify diskSize,
	// or the node group deployment will fail. For more information about using
	// launch templates with Amazon EKS, see Launch template support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	DiskSize int32 `json:"diskSize,omitempty"`

	// Specify the instance types for a node group. If you specify a GPU instance
	// type, be sure to specify AL2_x86_64_GPU with the amiType parameter. If you
	// specify launchTemplate, then you can specify zero or one instance type in
	// your launch template or you can specify 0-20 instance types for instanceTypes.
	// If however, you specify an instance type in your launch template and specify
	// any instanceTypes, the node group deployment will fail. If you don't specify
	// an instance type in a launch template or for instanceTypes, then t3.medium
	// is used, by default. If you specify Spot for capacityType, then we recommend
	// specifying multiple values for instanceTypes. For more information, see Managed
	// node group capacity types (https://docs.aws.amazon.com/eks/latest/userguide/managed-node-groups.html#managed-node-group-capacity-types)
	// and Launch template support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	InstanceTypes []string `json:"instanceTypes"`

	// The Kubernetes labels to be applied to the nodes in the node group when they
	// are created.
	Labels map[string]string `json:"labels,omitempty"`

	// The Amazon Resource Name (ARN) of the IAM role to associate with your node
	// group. The Amazon EKS worker node kubelet daemon makes calls to AWS APIs
	// on your behalf. Nodes receive permissions for these API calls through an
	// IAM instance profile and associated policies. Before you can launch nodes
	// and register them into a cluster, you must create an IAM role for those nodes
	// to use when they are launched. For more information, see Amazon EKS node
	// IAM role (https://docs.aws.amazon.com/eks/latest/userguide/worker_node_IAM_role.html)
	// in the Amazon EKS User Guide . If you specify launchTemplate, then don't
	// specify IamInstanceProfile (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_IamInstanceProfile.html)
	// in your launch template, or the node group deployment will fail. For more
	// information about using launch templates with Amazon EKS, see Launch template
	// support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	//
	// NodeRole is a required field
	NodeRole string `json:"nodeRole,omitempty"`

	// The subnets to use for the Auto Scaling group that is created for your node
	// group. These subnets must have the tag key kubernetes.io/cluster/CLUSTER_NAME
	// with a value of shared, where CLUSTER_NAME is replaced with the name of your
	// cluster. If you specify launchTemplate, then don't specify SubnetId (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CreateNetworkInterface.html)
	// in your launch template, or the node group deployment will fail. For more
	// information about using launch templates with Amazon EKS, see Launch template
	// support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	//
	// Subnets is a required field
	Subnets []string `json:"subnets"`

	// The metadata applied to the node group to assist with categorization and
	// organization. Each tag consists of a key and an optional value. You define
	// both. Node group tags do not propagate to any other resources associated
	// with the node group, such as the Amazon EC2 instances or subnets.
	Tags map[string]string `json:"tags,omitempty"`

	// The Kubernetes version to use for your managed nodes. By default, the Kubernetes
	// version of the cluster is used, and this is the only accepted specified value.
	// If you specify launchTemplate, and your launch template uses a custom AMI,
	// then don't specify version, or the node group deployment will fail. For more
	// information about using launch templates with Amazon EKS, see Launch template
	// support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	Version string `json:"version,omitempty"`

	// scaling config
	ScalingConfig *EKSNodegroupScalingConfig `json:"scalingConfig,omitempty"`
}

// Validate validates this e k s machine deployment cloud spec
func (m *EKSMachineDeploymentCloudSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCreatedAt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateScalingConfig(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *EKSMachineDeploymentCloudSpec) validateCreatedAt(formats strfmt.Registry) error {
	if swag.IsZero(m.CreatedAt) { // not required
		return nil
	}

	if err := validate.FormatOf("createdAt", "body", "date-time", m.CreatedAt.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *EKSMachineDeploymentCloudSpec) validateScalingConfig(formats strfmt.Registry) error {
	if swag.IsZero(m.ScalingConfig) { // not required
		return nil
	}

	if m.ScalingConfig != nil {
		if err := m.ScalingConfig.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("scalingConfig")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("scalingConfig")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this e k s machine deployment cloud spec based on the context it is used
func (m *EKSMachineDeploymentCloudSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateScalingConfig(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *EKSMachineDeploymentCloudSpec) contextValidateScalingConfig(ctx context.Context, formats strfmt.Registry) error {

	if m.ScalingConfig != nil {
		if err := m.ScalingConfig.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("scalingConfig")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("scalingConfig")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *EKSMachineDeploymentCloudSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *EKSMachineDeploymentCloudSpec) UnmarshalBinary(b []byte) error {
	var res EKSMachineDeploymentCloudSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
