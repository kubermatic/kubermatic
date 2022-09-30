// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"strconv"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// KubevirtNodeSpec KubevirtNodeSpec kubevirt specific node settings
//
// swagger:model KubevirtNodeSpec
type KubevirtNodeSpec struct {

	// CPUs states how many cpus the kubevirt node will have.
	// Required: true
	CPUs *string `json:"cpus"`

	// FlavorName states name of the virtual-machine flavor.
	FlavorName string `json:"flavorName,omitempty"`

	// FlavorProfile states name of virtual-machine profile.
	FlavorProfile string `json:"flavorProfile,omitempty"`

	// Memory states the memory that kubevirt node will have.
	// Required: true
	Memory *string `json:"memory"`

	// PodAffinityPreset describes pod affinity scheduling rules
	//
	// Deprecated: in favor of topology spread constraints
	PodAffinityPreset string `json:"podAffinityPreset,omitempty"`

	// PodAntiAffinityPreset describes pod anti-affinity scheduling rules
	//
	// Deprecated: in favor of topology spread constraints
	PodAntiAffinityPreset string `json:"podAntiAffinityPreset,omitempty"`

	// PrimaryDiskOSImage states the source from which the imported image will be downloaded.
	// This field contains:
	// a URL to download an Os Image from a HTTP source.
	// a DataVolume Name as source for DataVolume cloning.
	// Required: true
	PrimaryDiskOSImage *string `json:"primaryDiskOSImage"`

	// PrimaryDiskSize states the size of the provisioned pvc per node.
	// Required: true
	PrimaryDiskSize *string `json:"primaryDiskSize"`

	// PrimaryDiskStorageClassName states the storage class name for the provisioned PVCs.
	// Required: true
	PrimaryDiskStorageClassName *string `json:"primaryDiskStorageClassName"`

	// SecondaryDisks contains list of secondary-disks
	SecondaryDisks []*SecondaryDisks `json:"secondaryDisks"`

	// TopologySpreadConstraints describes topology spread constraints for VMs.
	TopologySpreadConstraints []*TopologySpreadConstraint `json:"topologySpreadConstraints"`

	// node affinity preset
	NodeAffinityPreset *NodeAffinityPreset `json:"nodeAffinityPreset,omitempty"`
}

// Validate validates this kubevirt node spec
func (m *KubevirtNodeSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCPUs(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMemory(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePrimaryDiskOSImage(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePrimaryDiskSize(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePrimaryDiskStorageClassName(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateSecondaryDisks(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateTopologySpreadConstraints(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNodeAffinityPreset(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *KubevirtNodeSpec) validateCPUs(formats strfmt.Registry) error {

	if err := validate.Required("cpus", "body", m.CPUs); err != nil {
		return err
	}

	return nil
}

func (m *KubevirtNodeSpec) validateMemory(formats strfmt.Registry) error {

	if err := validate.Required("memory", "body", m.Memory); err != nil {
		return err
	}

	return nil
}

func (m *KubevirtNodeSpec) validatePrimaryDiskOSImage(formats strfmt.Registry) error {

	if err := validate.Required("primaryDiskOSImage", "body", m.PrimaryDiskOSImage); err != nil {
		return err
	}

	return nil
}

func (m *KubevirtNodeSpec) validatePrimaryDiskSize(formats strfmt.Registry) error {

	if err := validate.Required("primaryDiskSize", "body", m.PrimaryDiskSize); err != nil {
		return err
	}

	return nil
}

func (m *KubevirtNodeSpec) validatePrimaryDiskStorageClassName(formats strfmt.Registry) error {

	if err := validate.Required("primaryDiskStorageClassName", "body", m.PrimaryDiskStorageClassName); err != nil {
		return err
	}

	return nil
}

func (m *KubevirtNodeSpec) validateSecondaryDisks(formats strfmt.Registry) error {
	if swag.IsZero(m.SecondaryDisks) { // not required
		return nil
	}

	for i := 0; i < len(m.SecondaryDisks); i++ {
		if swag.IsZero(m.SecondaryDisks[i]) { // not required
			continue
		}

		if m.SecondaryDisks[i] != nil {
			if err := m.SecondaryDisks[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("secondaryDisks" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("secondaryDisks" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *KubevirtNodeSpec) validateTopologySpreadConstraints(formats strfmt.Registry) error {
	if swag.IsZero(m.TopologySpreadConstraints) { // not required
		return nil
	}

	for i := 0; i < len(m.TopologySpreadConstraints); i++ {
		if swag.IsZero(m.TopologySpreadConstraints[i]) { // not required
			continue
		}

		if m.TopologySpreadConstraints[i] != nil {
			if err := m.TopologySpreadConstraints[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("topologySpreadConstraints" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("topologySpreadConstraints" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *KubevirtNodeSpec) validateNodeAffinityPreset(formats strfmt.Registry) error {
	if swag.IsZero(m.NodeAffinityPreset) { // not required
		return nil
	}

	if m.NodeAffinityPreset != nil {
		if err := m.NodeAffinityPreset.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("nodeAffinityPreset")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("nodeAffinityPreset")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this kubevirt node spec based on the context it is used
func (m *KubevirtNodeSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateSecondaryDisks(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateTopologySpreadConstraints(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateNodeAffinityPreset(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *KubevirtNodeSpec) contextValidateSecondaryDisks(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.SecondaryDisks); i++ {

		if m.SecondaryDisks[i] != nil {
			if err := m.SecondaryDisks[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("secondaryDisks" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("secondaryDisks" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *KubevirtNodeSpec) contextValidateTopologySpreadConstraints(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.TopologySpreadConstraints); i++ {

		if m.TopologySpreadConstraints[i] != nil {
			if err := m.TopologySpreadConstraints[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("topologySpreadConstraints" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("topologySpreadConstraints" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *KubevirtNodeSpec) contextValidateNodeAffinityPreset(ctx context.Context, formats strfmt.Registry) error {

	if m.NodeAffinityPreset != nil {
		if err := m.NodeAffinityPreset.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("nodeAffinityPreset")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("nodeAffinityPreset")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *KubevirtNodeSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *KubevirtNodeSpec) UnmarshalBinary(b []byte) error {
	var res KubevirtNodeSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
