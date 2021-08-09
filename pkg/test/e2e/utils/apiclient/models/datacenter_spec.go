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

// DatacenterSpec DatacenterSpec specifies the data for a datacenter.
//
// swagger:model DatacenterSpec
type DatacenterSpec struct {

	// Optional: Country of the seed as ISO-3166 two-letter code, e.g. DE or UK.
	// It is used for informational purposes.
	Country string `json:"country,omitempty"`

	// EnforceAuditLogging enforces audit logging on every cluster within the DC,
	// ignoring cluster-specific settings.
	EnforceAuditLogging bool `json:"enforceAuditLogging,omitempty"`

	// EnforcePodSecurityPolicy enforces pod security policy plugin on every clusters within the DC,
	// ignoring cluster-specific settings
	EnforcePodSecurityPolicy bool `json:"enforcePodSecurityPolicy,omitempty"`

	// Optional: Detailed location of the cluster, like "Hamburg" or "Datacenter 7".
	// It is used for informational purposes.
	Location string `json:"location,omitempty"`

	// Name of the datacenter provider. Extracted based on which provider is defined in the spec.
	// It is used for informational purposes.
	Provider string `json:"provider,omitempty"`

	// Deprecated. Automatically migrated to the RequiredEmailDomains field.
	RequiredEmailDomain string `json:"requiredEmailDomain,omitempty"`

	// required email domains
	RequiredEmailDomains []string `json:"requiredEmailDomains"`

	// Name of the seed this datacenter belongs to.
	Seed string `json:"seed,omitempty"`

	// alibaba
	Alibaba *DatacenterSpecAlibaba `json:"alibaba,omitempty"`

	// anexia
	Anexia *DatacenterSpecAnexia `json:"anexia,omitempty"`

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
	Kubevirt *DatacenterSpecKubevirt `json:"kubevirt,omitempty"`

	// node
	Node *NodeSettings `json:"node,omitempty"`

	// openstack
	Openstack *DatacenterSpecOpenstack `json:"openstack,omitempty"`

	// packet
	Packet *DatacenterSpecPacket `json:"packet,omitempty"`

	// vsphere
	Vsphere *DatacenterSpecVSphere `json:"vsphere,omitempty"`
}

// Validate validates this datacenter spec
func (m *DatacenterSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAlibaba(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateAnexia(formats); err != nil {
		res = append(res, err)
	}

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

	if err := m.validateKubevirt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNode(formats); err != nil {
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

func (m *DatacenterSpec) validateAlibaba(formats strfmt.Registry) error {
	if swag.IsZero(m.Alibaba) { // not required
		return nil
	}

	if m.Alibaba != nil {
		if err := m.Alibaba.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("alibaba")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) validateAnexia(formats strfmt.Registry) error {
	if swag.IsZero(m.Anexia) { // not required
		return nil
	}

	if m.Anexia != nil {
		if err := m.Anexia.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("anexia")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) validateAws(formats strfmt.Registry) error {
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

func (m *DatacenterSpec) validateAzure(formats strfmt.Registry) error {
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

func (m *DatacenterSpec) validateDigitalocean(formats strfmt.Registry) error {
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

func (m *DatacenterSpec) validateFake(formats strfmt.Registry) error {
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

func (m *DatacenterSpec) validateGcp(formats strfmt.Registry) error {
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

func (m *DatacenterSpec) validateHetzner(formats strfmt.Registry) error {
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

func (m *DatacenterSpec) validateKubevirt(formats strfmt.Registry) error {
	if swag.IsZero(m.Kubevirt) { // not required
		return nil
	}

	if m.Kubevirt != nil {
		if err := m.Kubevirt.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kubevirt")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) validateNode(formats strfmt.Registry) error {
	if swag.IsZero(m.Node) { // not required
		return nil
	}

	if m.Node != nil {
		if err := m.Node.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("node")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) validateOpenstack(formats strfmt.Registry) error {
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

func (m *DatacenterSpec) validatePacket(formats strfmt.Registry) error {
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

func (m *DatacenterSpec) validateVsphere(formats strfmt.Registry) error {
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

// ContextValidate validate this datacenter spec based on the context it is used
func (m *DatacenterSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateAlibaba(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateAnexia(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateAws(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateAzure(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateDigitalocean(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateFake(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateGcp(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateHetzner(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateKubevirt(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateNode(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateOpenstack(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidatePacket(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateVsphere(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *DatacenterSpec) contextValidateAlibaba(ctx context.Context, formats strfmt.Registry) error {

	if m.Alibaba != nil {
		if err := m.Alibaba.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("alibaba")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidateAnexia(ctx context.Context, formats strfmt.Registry) error {

	if m.Anexia != nil {
		if err := m.Anexia.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("anexia")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidateAws(ctx context.Context, formats strfmt.Registry) error {

	if m.Aws != nil {
		if err := m.Aws.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("aws")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidateAzure(ctx context.Context, formats strfmt.Registry) error {

	if m.Azure != nil {
		if err := m.Azure.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("azure")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidateDigitalocean(ctx context.Context, formats strfmt.Registry) error {

	if m.Digitalocean != nil {
		if err := m.Digitalocean.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("digitalocean")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidateFake(ctx context.Context, formats strfmt.Registry) error {

	if m.Fake != nil {
		if err := m.Fake.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("fake")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidateGcp(ctx context.Context, formats strfmt.Registry) error {

	if m.Gcp != nil {
		if err := m.Gcp.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("gcp")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidateHetzner(ctx context.Context, formats strfmt.Registry) error {

	if m.Hetzner != nil {
		if err := m.Hetzner.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("hetzner")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidateKubevirt(ctx context.Context, formats strfmt.Registry) error {

	if m.Kubevirt != nil {
		if err := m.Kubevirt.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kubevirt")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidateNode(ctx context.Context, formats strfmt.Registry) error {

	if m.Node != nil {
		if err := m.Node.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("node")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidateOpenstack(ctx context.Context, formats strfmt.Registry) error {

	if m.Openstack != nil {
		if err := m.Openstack.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("openstack")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidatePacket(ctx context.Context, formats strfmt.Registry) error {

	if m.Packet != nil {
		if err := m.Packet.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("packet")
			}
			return err
		}
	}

	return nil
}

func (m *DatacenterSpec) contextValidateVsphere(ctx context.Context, formats strfmt.Registry) error {

	if m.Vsphere != nil {
		if err := m.Vsphere.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vsphere")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *DatacenterSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *DatacenterSpec) UnmarshalBinary(b []byte) error {
	var res DatacenterSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
