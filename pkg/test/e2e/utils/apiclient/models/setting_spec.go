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

// SettingSpec setting spec
//
// swagger:model SettingSpec
type SettingSpec struct {

	// default node count
	DefaultNodeCount int8 `json:"defaultNodeCount,omitempty"`

	// display API docs
	DisplayAPIDocs bool `json:"displayAPIDocs,omitempty"`

	// display demo info
	DisplayDemoInfo bool `json:"displayDemoInfo,omitempty"`

	// display terms of service
	DisplayTermsOfService bool `json:"displayTermsOfService,omitempty"`

	// enable dashboard
	EnableDashboard bool `json:"enableDashboard,omitempty"`

	// enable external cluster import
	EnableExternalClusterImport bool `json:"enableExternalClusterImport,omitempty"`

	// enable o ID c kubeconfig
	EnableOIDCKubeconfig bool `json:"enableOIDCKubeconfig,omitempty"`

	// mla alertmanager prefix
	MlaAlertmanagerPrefix string `json:"mlaAlertmanagerPrefix,omitempty"`

	// mla grafana prefix
	MlaGrafanaPrefix string `json:"mlaGrafanaPrefix,omitempty"`

	// restrict project creation
	RestrictProjectCreation bool `json:"restrictProjectCreation,omitempty"`

	// user projects limit
	UserProjectsLimit int64 `json:"userProjectsLimit,omitempty"`

	// cleanup options
	CleanupOptions *CleanupOptions `json:"cleanupOptions,omitempty"`

	// custom links
	CustomLinks CustomLinks `json:"customLinks,omitempty"`

	// machine deployment VM resource quota
	MachineDeploymentVMResourceQuota *MachineDeploymentVMResourceQuota `json:"machineDeploymentVMResourceQuota,omitempty"`

	// mla options
	MlaOptions *MlaOptions `json:"mlaOptions,omitempty"`

	// opa options
	OpaOptions *OpaOptions `json:"opaOptions,omitempty"`
}

// Validate validates this setting spec
func (m *SettingSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCleanupOptions(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateCustomLinks(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMachineDeploymentVMResourceQuota(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMlaOptions(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOpaOptions(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *SettingSpec) validateCleanupOptions(formats strfmt.Registry) error {
	if swag.IsZero(m.CleanupOptions) { // not required
		return nil
	}

	if m.CleanupOptions != nil {
		if err := m.CleanupOptions.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("cleanupOptions")
			}
			return err
		}
	}

	return nil
}

func (m *SettingSpec) validateCustomLinks(formats strfmt.Registry) error {
	if swag.IsZero(m.CustomLinks) { // not required
		return nil
	}

	if err := m.CustomLinks.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("customLinks")
		}
		return err
	}

	return nil
}

func (m *SettingSpec) validateMachineDeploymentVMResourceQuota(formats strfmt.Registry) error {
	if swag.IsZero(m.MachineDeploymentVMResourceQuota) { // not required
		return nil
	}

	if m.MachineDeploymentVMResourceQuota != nil {
		if err := m.MachineDeploymentVMResourceQuota.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("machineDeploymentVMResourceQuota")
			}
			return err
		}
	}

	return nil
}

func (m *SettingSpec) validateMlaOptions(formats strfmt.Registry) error {
	if swag.IsZero(m.MlaOptions) { // not required
		return nil
	}

	if m.MlaOptions != nil {
		if err := m.MlaOptions.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("mlaOptions")
			}
			return err
		}
	}

	return nil
}

func (m *SettingSpec) validateOpaOptions(formats strfmt.Registry) error {
	if swag.IsZero(m.OpaOptions) { // not required
		return nil
	}

	if m.OpaOptions != nil {
		if err := m.OpaOptions.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("opaOptions")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this setting spec based on the context it is used
func (m *SettingSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateCleanupOptions(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateCustomLinks(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateMachineDeploymentVMResourceQuota(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateMlaOptions(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateOpaOptions(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *SettingSpec) contextValidateCleanupOptions(ctx context.Context, formats strfmt.Registry) error {

	if m.CleanupOptions != nil {
		if err := m.CleanupOptions.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("cleanupOptions")
			}
			return err
		}
	}

	return nil
}

func (m *SettingSpec) contextValidateCustomLinks(ctx context.Context, formats strfmt.Registry) error {

	if err := m.CustomLinks.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("customLinks")
		}
		return err
	}

	return nil
}

func (m *SettingSpec) contextValidateMachineDeploymentVMResourceQuota(ctx context.Context, formats strfmt.Registry) error {

	if m.MachineDeploymentVMResourceQuota != nil {
		if err := m.MachineDeploymentVMResourceQuota.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("machineDeploymentVMResourceQuota")
			}
			return err
		}
	}

	return nil
}

func (m *SettingSpec) contextValidateMlaOptions(ctx context.Context, formats strfmt.Registry) error {

	if m.MlaOptions != nil {
		if err := m.MlaOptions.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("mlaOptions")
			}
			return err
		}
	}

	return nil
}

func (m *SettingSpec) contextValidateOpaOptions(ctx context.Context, formats strfmt.Registry) error {

	if m.OpaOptions != nil {
		if err := m.OpaOptions.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("opaOptions")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *SettingSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *SettingSpec) UnmarshalBinary(b []byte) error {
	var res SettingSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
