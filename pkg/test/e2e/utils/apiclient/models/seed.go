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

// Seed Seed represents a seed object
//
// swagger:model Seed
type Seed struct {

	// Optional: Country of the seed as ISO-3166 two-letter code, e.g. DE or UK.
	// For informational purposes in the Kubermatic dashboard only.
	Country string `json:"country,omitempty"`

	// Optional: Detailed location of the cluster, like "Hamburg" or "Datacenter 7".
	// For informational purposes in the Kubermatic dashboard only.
	Location string `json:"location,omitempty"`

	// Name represents human readable name for the resource
	Name string `json:"name,omitempty"`

	// Optional: This can be used to override the DNS name used for this seed.
	// By default the seed name is used.
	SeedDNSOverwrite string `json:"seed_dns_overwrite,omitempty"`

	// Datacenters contains a map of the possible datacenters (DCs) in this seed.
	// Each DC must have a globally unique identifier (i.e. names must be unique
	// across all seeds).
	SeedDatacenters map[string]Datacenter `json:"datacenters,omitempty"`

	// backup restore
	BackupRestore *BackupDestination `json:"backupRestore,omitempty"`

	// expose strategy
	ExposeStrategy ExposeStrategy `json:"expose_strategy,omitempty"`

	// kubeconfig
	Kubeconfig *ObjectReference `json:"kubeconfig,omitempty"`

	// mla
	Mla *SeedMLASettings `json:"mla,omitempty"`

	// proxy settings
	ProxySettings *ProxySettings `json:"proxy_settings,omitempty"`
}

// Validate validates this seed
func (m *Seed) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateSeedDatacenters(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateBackupRestore(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateExposeStrategy(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateKubeconfig(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMla(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateProxySettings(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Seed) validateSeedDatacenters(formats strfmt.Registry) error {
	if swag.IsZero(m.SeedDatacenters) { // not required
		return nil
	}

	for k := range m.SeedDatacenters {

		if err := validate.Required("datacenters"+"."+k, "body", m.SeedDatacenters[k]); err != nil {
			return err
		}
		if val, ok := m.SeedDatacenters[k]; ok {
			if err := val.Validate(formats); err != nil {
				return err
			}
		}

	}

	return nil
}

func (m *Seed) validateBackupRestore(formats strfmt.Registry) error {
	if swag.IsZero(m.BackupRestore) { // not required
		return nil
	}

	if m.BackupRestore != nil {
		if err := m.BackupRestore.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("backupRestore")
			}
			return err
		}
	}

	return nil
}

func (m *Seed) validateExposeStrategy(formats strfmt.Registry) error {
	if swag.IsZero(m.ExposeStrategy) { // not required
		return nil
	}

	if err := m.ExposeStrategy.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("expose_strategy")
		}
		return err
	}

	return nil
}

func (m *Seed) validateKubeconfig(formats strfmt.Registry) error {
	if swag.IsZero(m.Kubeconfig) { // not required
		return nil
	}

	if m.Kubeconfig != nil {
		if err := m.Kubeconfig.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kubeconfig")
			}
			return err
		}
	}

	return nil
}

func (m *Seed) validateMla(formats strfmt.Registry) error {
	if swag.IsZero(m.Mla) { // not required
		return nil
	}

	if m.Mla != nil {
		if err := m.Mla.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("mla")
			}
			return err
		}
	}

	return nil
}

func (m *Seed) validateProxySettings(formats strfmt.Registry) error {
	if swag.IsZero(m.ProxySettings) { // not required
		return nil
	}

	if m.ProxySettings != nil {
		if err := m.ProxySettings.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("proxy_settings")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this seed based on the context it is used
func (m *Seed) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateSeedDatacenters(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateBackupRestore(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateExposeStrategy(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateKubeconfig(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateMla(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateProxySettings(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Seed) contextValidateSeedDatacenters(ctx context.Context, formats strfmt.Registry) error {

	for k := range m.SeedDatacenters {

		if val, ok := m.SeedDatacenters[k]; ok {
			if err := val.ContextValidate(ctx, formats); err != nil {
				return err
			}
		}

	}

	return nil
}

func (m *Seed) contextValidateBackupRestore(ctx context.Context, formats strfmt.Registry) error {

	if m.BackupRestore != nil {
		if err := m.BackupRestore.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("backupRestore")
			}
			return err
		}
	}

	return nil
}

func (m *Seed) contextValidateExposeStrategy(ctx context.Context, formats strfmt.Registry) error {

	if err := m.ExposeStrategy.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("expose_strategy")
		}
		return err
	}

	return nil
}

func (m *Seed) contextValidateKubeconfig(ctx context.Context, formats strfmt.Registry) error {

	if m.Kubeconfig != nil {
		if err := m.Kubeconfig.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kubeconfig")
			}
			return err
		}
	}

	return nil
}

func (m *Seed) contextValidateMla(ctx context.Context, formats strfmt.Registry) error {

	if m.Mla != nil {
		if err := m.Mla.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("mla")
			}
			return err
		}
	}

	return nil
}

func (m *Seed) contextValidateProxySettings(ctx context.Context, formats strfmt.Registry) error {

	if m.ProxySettings != nil {
		if err := m.ProxySettings.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("proxy_settings")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *Seed) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Seed) UnmarshalBinary(b []byte) error {
	var res Seed
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
