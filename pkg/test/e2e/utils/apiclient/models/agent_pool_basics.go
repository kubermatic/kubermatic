// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// AgentPoolBasics agent pool basics
//
// swagger:model AgentPoolBasics
type AgentPoolBasics struct {

	// AvailabilityZones - The list of Availability zones to use for nodes. This can only be specified if the AgentPoolType property is 'VirtualMachineScaleSets'.
	AvailabilityZones []string `json:"availabilityZones"`

	// Count - Number of agents (VMs) to host docker containers. Allowed values must be in the range of 0 to 1000 (inclusive) for user pools and in the range of 1 to 1000 (inclusive) for system pools. The default value is 1.
	Count int32 `json:"count,omitempty"`

	// EnableAutoScaling - Whether to enable auto-scaler
	EnableAutoScaling bool `json:"enableAutoScaling,omitempty"`

	// MaxCount - The maximum number of nodes for auto-scaling
	MaxCount int32 `json:"maxCount,omitempty"`

	// MinCount - The minimum number of nodes for auto-scaling
	MinCount int32 `json:"minCount,omitempty"`

	// Mode - Possible values include: 'System', 'User'
	Mode string `json:"mode,omitempty"`

	// OrchestratorVersion - As a best practice, you should upgrade all node pools in an AKS cluster to the same Kubernetes version. The node pool version must have the same major version as the control plane. The node pool minor version must be within two minor versions of the control plane version. The node pool version cannot be greater than the control plane version. For more information see [upgrading a node pool](https://docs.microsoft.com/azure/aks/use-multiple-node-pools#upgrade-a-node-pool).
	OrchestratorVersion string `json:"orchestratorVersion,omitempty"`

	// OsType - Possible values include: 'Linux', 'Windows'
	OsType string `json:"osType,omitempty"`

	// VMSize - VM size availability varies by region. If a node contains insufficient compute resources (memory, cpu, etc) pods might fail to run correctly. For more details on restricted VM sizes, see: https://docs.microsoft.com/azure/aks/quotas-skus-regions
	VMSize string `json:"vmSize,omitempty"`
}

// Validate validates this agent pool basics
func (m *AgentPoolBasics) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this agent pool basics based on context it is used
func (m *AgentPoolBasics) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AgentPoolBasics) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AgentPoolBasics) UnmarshalBinary(b []byte) error {
	var res AgentPoolBasics
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
