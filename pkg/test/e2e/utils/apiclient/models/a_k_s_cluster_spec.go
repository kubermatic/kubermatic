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

// AKSClusterSpec AKSClusterSpec Azure Kubernetes Service cluster.
//
// swagger:model AKSClusterSpec
type AKSClusterSpec struct {

	// The timestamp of resource creation (UTC).
	// Format: date-time
	CreatedAt strfmt.DateTime `json:"createdAt,omitempty"`

	// The identity that created the resource.
	CreatedBy string `json:"createdBy,omitempty"`

	// DNSPrefix - This cannot be updated once the Managed Cluster has been created.
	DNSPrefix string `json:"dnsPrefix,omitempty"`

	// EnableRBAC - Whether Kubernetes Role-Based Access Control Enabled.
	EnableRBAC bool `json:"enableRBAC,omitempty"`

	// Fqdn - READ-ONLY; The FQDN of the master pool.
	Fqdn string `json:"fqdn,omitempty"`

	// FqdnSubdomain - This cannot be updated once the Managed Cluster has been created.
	FqdnSubdomain string `json:"fqdnSubdomain,omitempty"`

	// KubernetesVersion - When you upgrade a supported AKS cluster, Kubernetes minor versions cannot be skipped. All upgrades must be performed sequentially by major version number. For example, upgrades between 1.14.x -> 1.15.x or 1.15.x -> 1.16.x are allowed, however 1.14.x -> 1.16.x is not allowed. See [upgrading an AKS cluster](https://docs.microsoft.com/azure/aks/upgrade-cluster) for more details.
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`

	// PrivateFQDN - READ-ONLY; The FQDN of private cluster.
	PrivateFQDN string `json:"privateFQDN,omitempty"`

	// Resource tags.
	Tags map[string]string `json:"tags,omitempty"`

	// machine deployment spec
	MachineDeploymentSpec *AKSMachineDeploymentCloudSpec `json:"machineDeploymentSpec,omitempty"`

	// network profile
	NetworkProfile *AKSNetworkProfile `json:"networkProfile,omitempty"`
}

// Validate validates this a k s cluster spec
func (m *AKSClusterSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCreatedAt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMachineDeploymentSpec(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNetworkProfile(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *AKSClusterSpec) validateCreatedAt(formats strfmt.Registry) error {
	if swag.IsZero(m.CreatedAt) { // not required
		return nil
	}

	if err := validate.FormatOf("createdAt", "body", "date-time", m.CreatedAt.String(), formats); err != nil {
		return err
	}

	return nil
}

func (m *AKSClusterSpec) validateMachineDeploymentSpec(formats strfmt.Registry) error {
	if swag.IsZero(m.MachineDeploymentSpec) { // not required
		return nil
	}

	if m.MachineDeploymentSpec != nil {
		if err := m.MachineDeploymentSpec.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("machineDeploymentSpec")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("machineDeploymentSpec")
			}
			return err
		}
	}

	return nil
}

func (m *AKSClusterSpec) validateNetworkProfile(formats strfmt.Registry) error {
	if swag.IsZero(m.NetworkProfile) { // not required
		return nil
	}

	if m.NetworkProfile != nil {
		if err := m.NetworkProfile.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("networkProfile")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("networkProfile")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this a k s cluster spec based on the context it is used
func (m *AKSClusterSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateMachineDeploymentSpec(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateNetworkProfile(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *AKSClusterSpec) contextValidateMachineDeploymentSpec(ctx context.Context, formats strfmt.Registry) error {

	if m.MachineDeploymentSpec != nil {
		if err := m.MachineDeploymentSpec.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("machineDeploymentSpec")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("machineDeploymentSpec")
			}
			return err
		}
	}

	return nil
}

func (m *AKSClusterSpec) contextValidateNetworkProfile(ctx context.Context, formats strfmt.Registry) error {

	if m.NetworkProfile != nil {
		if err := m.NetworkProfile.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("networkProfile")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("networkProfile")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *AKSClusterSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AKSClusterSpec) UnmarshalBinary(b []byte) error {
	var res AKSClusterSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
