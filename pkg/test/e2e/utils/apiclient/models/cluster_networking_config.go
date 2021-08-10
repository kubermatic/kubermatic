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

// ClusterNetworkingConfig ClusterNetworkingConfig specifies the different networking
// parameters for a cluster.
//
// swagger:model ClusterNetworkingConfig
type ClusterNetworkingConfig struct {

	// Domain name for services.
	DNSDomain string `json:"dnsDomain,omitempty"`

	// NodeLocalDNSCacheEnabled controls whether the NodeLocal DNS Cache feature is enabled.
	// Defaults to true.
	NodeLocalDNSCacheEnabled bool `json:"nodeLocalDNSCacheEnabled,omitempty"`

	// ProxyMode defines the kube-proxy mode (ipvs/iptables).
	// Defaults to ipvs.
	ProxyMode string `json:"proxyMode,omitempty"`

	// ipvs
	Ipvs *IPVSConfiguration `json:"ipvs,omitempty"`

	// pods
	Pods *NetworkRanges `json:"pods,omitempty"`

	// services
	Services *NetworkRanges `json:"services,omitempty"`
}

// Validate validates this cluster networking config
func (m *ClusterNetworkingConfig) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateIpvs(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePods(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateServices(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ClusterNetworkingConfig) validateIpvs(formats strfmt.Registry) error {
	if swag.IsZero(m.Ipvs) { // not required
		return nil
	}

	if m.Ipvs != nil {
		if err := m.Ipvs.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ipvs")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterNetworkingConfig) validatePods(formats strfmt.Registry) error {
	if swag.IsZero(m.Pods) { // not required
		return nil
	}

	if m.Pods != nil {
		if err := m.Pods.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("pods")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterNetworkingConfig) validateServices(formats strfmt.Registry) error {
	if swag.IsZero(m.Services) { // not required
		return nil
	}

	if m.Services != nil {
		if err := m.Services.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("services")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this cluster networking config based on the context it is used
func (m *ClusterNetworkingConfig) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateIpvs(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidatePods(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateServices(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ClusterNetworkingConfig) contextValidateIpvs(ctx context.Context, formats strfmt.Registry) error {

	if m.Ipvs != nil {
		if err := m.Ipvs.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("ipvs")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterNetworkingConfig) contextValidatePods(ctx context.Context, formats strfmt.Registry) error {

	if m.Pods != nil {
		if err := m.Pods.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("pods")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterNetworkingConfig) contextValidateServices(ctx context.Context, formats strfmt.Registry) error {

	if m.Services != nil {
		if err := m.Services.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("services")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ClusterNetworkingConfig) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ClusterNetworkingConfig) UnmarshalBinary(b []byte) error {
	var res ClusterNetworkingConfig
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
