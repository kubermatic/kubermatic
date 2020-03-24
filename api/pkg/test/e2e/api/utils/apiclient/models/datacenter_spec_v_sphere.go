// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// DatacenterSpecVSphere DatacenterSpecVSphere describes a vSphere datacenter
//
// swagger:model DatacenterSpecVSphere
type DatacenterSpecVSphere struct {

	// If set to true, disables the TLS certificate check against the endpoint.
	AllowInsecure bool `json:"allow_insecure,omitempty"`

	// The name of the Kubernetes cluster to use.
	Cluster string `json:"cluster,omitempty"`

	// The name of the datacenter to use.
	Datacenter string `json:"datacenter,omitempty"`

	// The name of the datastore to use.
	Datastore string `json:"datastore,omitempty"`

	// Endpoint URL to use, including protocol, for example "https://vcenter.example.com".
	Endpoint string `json:"endpoint,omitempty"`

	// Optional: The root path for cluster specific VM folders. Each cluster gets its own
	// folder below the root folder. Must be the FQDN (for example
	// "/datacenter-1/vm/all-kubermatic-vms-in-here") and defaults to the root VM
	// folder: "/datacenter-1/vm"
	RootPath string `json:"root_path,omitempty"`

	// infra management user
	InfraManagementUser *VSphereCredentials `json:"infra_management_user,omitempty"`

	// templates
	Templates ImageList `json:"templates,omitempty"`
}

// Validate validates this datacenter spec v sphere
func (m *DatacenterSpecVSphere) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateInfraManagementUser(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateTemplates(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *DatacenterSpecVSphere) validateInfraManagementUser(formats strfmt.Registry) error {

	if swag.IsZero(m.InfraManagementUser) { // not required
		return nil
	}

	if m.InfraManagementUser != nil {
		if err := m.InfraManagementUser.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("infra_management_user")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpecVSphere) validateTemplates(formats strfmt.Registry) error {

	if swag.IsZero(m.Templates) { // not required
		return nil
	}

	if err := m.Templates.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("templates")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *DatacenterSpecVSphere) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *DatacenterSpecVSphere) UnmarshalBinary(b []byte) error {
	var res DatacenterSpecVSphere
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
