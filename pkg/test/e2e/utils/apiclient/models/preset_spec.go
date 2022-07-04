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

// PresetSpec Presets specifies default presets for supported providers.
//
// swagger:model PresetSpec
type PresetSpec struct {

	// enabled
	Enabled bool `json:"enabled,omitempty"`

	// required emails
	RequiredEmails []string `json:"requiredEmails"`

	// aks
	Aks *AKS `json:"aks,omitempty"`

	// alibaba
	Alibaba *Alibaba `json:"alibaba,omitempty"`

	// anexia
	Anexia *Anexia `json:"anexia,omitempty"`

	// aws
	Aws *AWS `json:"aws,omitempty"`

	// azure
	Azure *Azure `json:"azure,omitempty"`

	// digitalocean
	Digitalocean *Digitalocean `json:"digitalocean,omitempty"`

	// eks
	Eks *EKS `json:"eks,omitempty"`

	// fake
	Fake *Fake `json:"fake,omitempty"`

	// gcp
	Gcp *GCP `json:"gcp,omitempty"`

	// gke
	Gke *GKE `json:"gke,omitempty"`

	// hetzner
	Hetzner *Hetzner `json:"hetzner,omitempty"`

	// kubevirt
	Kubevirt *Kubevirt `json:"kubevirt,omitempty"`

	// nutanix
	Nutanix *Nutanix `json:"nutanix,omitempty"`

	// openstack
	Openstack *Openstack `json:"openstack,omitempty"`

	// packet
	Packet *Packet `json:"packet,omitempty"`

	// vmwareclouddirector
	Vmwareclouddirector *VMwareCloudDirector `json:"vmwareclouddirector,omitempty"`

	// vsphere
	Vsphere *VSphere `json:"vsphere,omitempty"`
}

// Validate validates this preset spec
func (m *PresetSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAks(formats); err != nil {
		res = append(res, err)
	}

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

	if err := m.validateEks(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateFake(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateGcp(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateGke(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateHetzner(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateKubevirt(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNutanix(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOpenstack(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validatePacket(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVmwareclouddirector(formats); err != nil {
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

func (m *PresetSpec) validateAks(formats strfmt.Registry) error {
	if swag.IsZero(m.Aks) { // not required
		return nil
	}

	if m.Aks != nil {
		if err := m.Aks.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("aks")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("aks")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateAlibaba(formats strfmt.Registry) error {
	if swag.IsZero(m.Alibaba) { // not required
		return nil
	}

	if m.Alibaba != nil {
		if err := m.Alibaba.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("alibaba")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("alibaba")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateAnexia(formats strfmt.Registry) error {
	if swag.IsZero(m.Anexia) { // not required
		return nil
	}

	if m.Anexia != nil {
		if err := m.Anexia.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("anexia")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("anexia")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateAws(formats strfmt.Registry) error {
	if swag.IsZero(m.Aws) { // not required
		return nil
	}

	if m.Aws != nil {
		if err := m.Aws.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("aws")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("aws")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateAzure(formats strfmt.Registry) error {
	if swag.IsZero(m.Azure) { // not required
		return nil
	}

	if m.Azure != nil {
		if err := m.Azure.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("azure")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("azure")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateDigitalocean(formats strfmt.Registry) error {
	if swag.IsZero(m.Digitalocean) { // not required
		return nil
	}

	if m.Digitalocean != nil {
		if err := m.Digitalocean.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("digitalocean")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("digitalocean")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateEks(formats strfmt.Registry) error {
	if swag.IsZero(m.Eks) { // not required
		return nil
	}

	if m.Eks != nil {
		if err := m.Eks.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("eks")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("eks")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateFake(formats strfmt.Registry) error {
	if swag.IsZero(m.Fake) { // not required
		return nil
	}

	if m.Fake != nil {
		if err := m.Fake.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("fake")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("fake")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateGcp(formats strfmt.Registry) error {
	if swag.IsZero(m.Gcp) { // not required
		return nil
	}

	if m.Gcp != nil {
		if err := m.Gcp.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("gcp")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("gcp")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateGke(formats strfmt.Registry) error {
	if swag.IsZero(m.Gke) { // not required
		return nil
	}

	if m.Gke != nil {
		if err := m.Gke.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("gke")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("gke")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateHetzner(formats strfmt.Registry) error {
	if swag.IsZero(m.Hetzner) { // not required
		return nil
	}

	if m.Hetzner != nil {
		if err := m.Hetzner.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("hetzner")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("hetzner")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateKubevirt(formats strfmt.Registry) error {
	if swag.IsZero(m.Kubevirt) { // not required
		return nil
	}

	if m.Kubevirt != nil {
		if err := m.Kubevirt.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kubevirt")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("kubevirt")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateNutanix(formats strfmt.Registry) error {
	if swag.IsZero(m.Nutanix) { // not required
		return nil
	}

	if m.Nutanix != nil {
		if err := m.Nutanix.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("nutanix")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("nutanix")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateOpenstack(formats strfmt.Registry) error {
	if swag.IsZero(m.Openstack) { // not required
		return nil
	}

	if m.Openstack != nil {
		if err := m.Openstack.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("openstack")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("openstack")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validatePacket(formats strfmt.Registry) error {
	if swag.IsZero(m.Packet) { // not required
		return nil
	}

	if m.Packet != nil {
		if err := m.Packet.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("packet")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("packet")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateVmwareclouddirector(formats strfmt.Registry) error {
	if swag.IsZero(m.Vmwareclouddirector) { // not required
		return nil
	}

	if m.Vmwareclouddirector != nil {
		if err := m.Vmwareclouddirector.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vmwareclouddirector")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("vmwareclouddirector")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) validateVsphere(formats strfmt.Registry) error {
	if swag.IsZero(m.Vsphere) { // not required
		return nil
	}

	if m.Vsphere != nil {
		if err := m.Vsphere.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vsphere")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("vsphere")
			}
			return err
		}
	}

	return nil
}

// ContextValidate validate this preset spec based on the context it is used
func (m *PresetSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateAks(ctx, formats); err != nil {
		res = append(res, err)
	}

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

	if err := m.contextValidateEks(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateFake(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateGcp(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateGke(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateHetzner(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateKubevirt(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateNutanix(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateOpenstack(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidatePacket(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateVmwareclouddirector(ctx, formats); err != nil {
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

func (m *PresetSpec) contextValidateAks(ctx context.Context, formats strfmt.Registry) error {

	if m.Aks != nil {
		if err := m.Aks.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("aks")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("aks")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateAlibaba(ctx context.Context, formats strfmt.Registry) error {

	if m.Alibaba != nil {
		if err := m.Alibaba.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("alibaba")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("alibaba")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateAnexia(ctx context.Context, formats strfmt.Registry) error {

	if m.Anexia != nil {
		if err := m.Anexia.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("anexia")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("anexia")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateAws(ctx context.Context, formats strfmt.Registry) error {

	if m.Aws != nil {
		if err := m.Aws.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("aws")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("aws")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateAzure(ctx context.Context, formats strfmt.Registry) error {

	if m.Azure != nil {
		if err := m.Azure.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("azure")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("azure")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateDigitalocean(ctx context.Context, formats strfmt.Registry) error {

	if m.Digitalocean != nil {
		if err := m.Digitalocean.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("digitalocean")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("digitalocean")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateEks(ctx context.Context, formats strfmt.Registry) error {

	if m.Eks != nil {
		if err := m.Eks.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("eks")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("eks")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateFake(ctx context.Context, formats strfmt.Registry) error {

	if m.Fake != nil {
		if err := m.Fake.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("fake")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("fake")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateGcp(ctx context.Context, formats strfmt.Registry) error {

	if m.Gcp != nil {
		if err := m.Gcp.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("gcp")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("gcp")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateGke(ctx context.Context, formats strfmt.Registry) error {

	if m.Gke != nil {
		if err := m.Gke.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("gke")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("gke")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateHetzner(ctx context.Context, formats strfmt.Registry) error {

	if m.Hetzner != nil {
		if err := m.Hetzner.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("hetzner")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("hetzner")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateKubevirt(ctx context.Context, formats strfmt.Registry) error {

	if m.Kubevirt != nil {
		if err := m.Kubevirt.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kubevirt")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("kubevirt")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateNutanix(ctx context.Context, formats strfmt.Registry) error {

	if m.Nutanix != nil {
		if err := m.Nutanix.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("nutanix")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("nutanix")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateOpenstack(ctx context.Context, formats strfmt.Registry) error {

	if m.Openstack != nil {
		if err := m.Openstack.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("openstack")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("openstack")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidatePacket(ctx context.Context, formats strfmt.Registry) error {

	if m.Packet != nil {
		if err := m.Packet.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("packet")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("packet")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateVmwareclouddirector(ctx context.Context, formats strfmt.Registry) error {

	if m.Vmwareclouddirector != nil {
		if err := m.Vmwareclouddirector.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vmwareclouddirector")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("vmwareclouddirector")
			}
			return err
		}
	}

	return nil
}

func (m *PresetSpec) contextValidateVsphere(ctx context.Context, formats strfmt.Registry) error {

	if m.Vsphere != nil {
		if err := m.Vsphere.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("vsphere")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("vsphere")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *PresetSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *PresetSpec) UnmarshalBinary(b []byte) error {
	var res PresetSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
