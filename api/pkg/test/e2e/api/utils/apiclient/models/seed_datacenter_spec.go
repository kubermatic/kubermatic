// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	strfmt "github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// SeedDatacenterSpec SeedDatacenterSpec mutually points to provider datacenter spec
// swagger:model SeedDatacenterSpec
type SeedDatacenterSpec struct {

	// EnforceAuditLogging enforces audit logging on every cluster within the DC,
	// ignoring cluster-specific settings.
	EnforceAuditLogging bool `json:"enforceAuditLogging,omitempty"`

	// Optional: When defined, only users with an e-mail address on the
	// given domains can make use of this datacenter. You can define multiple
	// domains, e.g. "example.com", one of which must match the email domain
	// exactly (i.e. "example.com" will not match "user@test.example.com").
	// RequiredEmailDomain is deprecated. Automatically migrated to the RequiredEmailDomains field.
	RequiredEmailDomain string `json:"requiredEmailDomain,omitempty"`

	// required email domains
	RequiredEmailDomains []string `json:"requiredEmailDomains"`

	// aws
	Aws *DatacenterSpecAWS `json:"aws,omitempty"`

	// azure
	Azure *DatacenterSpecAzure `json:"azure,omitempty"`

	// bringyourown
	Bringyourown DatacenterSpecBringYourOwn `json:"bringyourown,omitempty"`

	// digitalocean
	Digitalocean *DatacenterSpecDigitalocean `json:"digitalocean,omitempty"`

	// fake
	Fake *DatacenterSpecFake `json:"fake,omitempty"`

	// gcp
	Gcp *DatacenterSpecGCP `json:"gcp,omitempty"`

	// hetzner
	Hetzner *DatacenterSpecHetzner `json:"hetzner,omitempty"`

	// kubevirt
	Kubevirt DatacenterSpecKubevirt `json:"kubevirt,omitempty"`

	// openstack
	Openstack *DatacenterSpecOpenstack `json:"openstack,omitempty"`

	// packet
	Packet *DatacenterSpecPacket `json:"packet,omitempty"`

	// vsphere
	Vsphere *DatacenterSpecVSphere `json:"vsphere,omitempty"`
}

// Validate validates this seed datacenter spec
func (m *SeedDatacenterSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAws(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateAzure(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateDigitalocean(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateFake(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateGcp(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateHetzner(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOpenstack(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePacket(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVsphere(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *SeedDatacenterSpec) validateAws(formats strfmt.Registry) error {

	if swag.IsZero(m.Aws) { // not required
		return nil
	}

	if m.Aws != nil {
		if err := m.Aws.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("aws")
			}
			return err
		}
	}

	return nil
}

func (m *SeedDatacenterSpec) validateAzure(formats strfmt.Registry) error {

	if swag.IsZero(m.Azure) { // not required
		return nil
	}

	if m.Azure != nil {
		if err := m.Azure.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("azure")
			}
			return err
		}
	}

	return nil
}

func (m *SeedDatacenterSpec) validateDigitalocean(formats strfmt.Registry) error {

	if swag.IsZero(m.Digitalocean) { // not required
		return nil
	}

	if m.Digitalocean != nil {
		if err := m.Digitalocean.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("digitalocean")
			}
			return err
		}
	}

	return nil
}

func (m *SeedDatacenterSpec) validateFake(formats strfmt.Registry) error {

	if swag.IsZero(m.Fake) { // not required
		return nil
	}

	if m.Fake != nil {
		if err := m.Fake.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("fake")
			}
			return err
		}
	}

	return nil
}

func (m *SeedDatacenterSpec) validateGcp(formats strfmt.Registry) error {

	if swag.IsZero(m.Gcp) { // not required
		return nil
	}

	if m.Gcp != nil {
		if err := m.Gcp.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("gcp")
			}
			return err
		}
	}

	return nil
}

func (m *SeedDatacenterSpec) validateHetzner(formats strfmt.Registry) error {

	if swag.IsZero(m.Hetzner) { // not required
		return nil
	}

	if m.Hetzner != nil {
		if err := m.Hetzner.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("hetzner")
			}
			return err
		}
	}

	return nil
}

func (m *SeedDatacenterSpec) validateOpenstack(formats strfmt.Registry) error {

	if swag.IsZero(m.Openstack) { // not required
		return nil
	}

	if m.Openstack != nil {
		if err := m.Openstack.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("openstack")
			}
			return err
		}
	}

	return nil
}

func (m *SeedDatacenterSpec) validatePacket(formats strfmt.Registry) error {

	if swag.IsZero(m.Packet) { // not required
		return nil
	}

	if m.Packet != nil {
		if err := m.Packet.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("packet")
			}
			return err
		}
	}

	return nil
}

func (m *SeedDatacenterSpec) validateVsphere(formats strfmt.Registry) error {

	if swag.IsZero(m.Vsphere) { // not required
		return nil
	}

	if m.Vsphere != nil {
		if err := m.Vsphere.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vsphere")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *SeedDatacenterSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *SeedDatacenterSpec) UnmarshalBinary(b []byte) error {
	var res SeedDatacenterSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
