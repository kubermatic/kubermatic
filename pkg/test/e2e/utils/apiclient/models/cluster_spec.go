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
)

// ClusterSpec ClusterSpec defines the cluster specification.
//
// swagger:model ClusterSpec
type ClusterSpec struct {

	// Additional Admission Controller plugins
	AdmissionPlugins []string `json:"admissionPlugins"`

	// ContainerRuntime to use, i.e. Docker or containerd. By default containerd will be used.
	ContainerRuntime string `json:"containerRuntime,omitempty"`

	// EnableOperatingSystemManager enables OSM which in-turn is responsible for creating and managing worker node configuration.
	EnableOperatingSystemManager bool `json:"enableOperatingSystemManager,omitempty"`

	// EnableUserSSHKeyAgent control whether the UserSSHKeyAgent will be deployed in the user cluster or not.
	// If it was enabled, the agent will be deployed and used to sync the user ssh keys, that the user attach
	// to the created cluster. If the agent was disabled, it won't be deployed in the user cluster, thus after
	// the cluster creation any attached ssh keys won't be synced to the worker nodes. Once the agent is enabled/disabled
	// it cannot be changed after the cluster is being created.
	EnableUserSSHKeyAgent bool `json:"enableUserSSHKeyAgent,omitempty"`

	// MachineNetworks optionally specifies the parameters for IPAM.
	MachineNetworks []*MachineNetworkingConfig `json:"machineNetworks"`

	// PodNodeSelectorAdmissionPluginConfig provides the configuration for the PodNodeSelector.
	// It's used by the backend to create a configuration file for this plugin.
	// The key:value from the map is converted to the namespace:<node-selectors-labels> in the file.
	// The format in a file:
	// podNodeSelectorPluginConfig:
	// clusterDefaultNodeSelector: <node-selectors-labels>
	// namespace1: <node-selectors-labels>
	// namespace2: <node-selectors-labels>
	PodNodeSelectorAdmissionPluginConfig map[string]string `json:"podNodeSelectorAdmissionPluginConfig,omitempty"`

	// If active the EventRateLimit admission plugin is configured at the apiserver
	UseEventRateLimitAdmissionPlugin bool `json:"useEventRateLimitAdmissionPlugin,omitempty"`

	// If active the PodNodeSelector admission plugin is configured at the apiserver
	UsePodNodeSelectorAdmissionPlugin bool `json:"usePodNodeSelectorAdmissionPlugin,omitempty"`

	// If active the PodSecurityPolicy admission plugin is configured at the apiserver
	UsePodSecurityPolicyAdmissionPlugin bool `json:"usePodSecurityPolicyAdmissionPlugin,omitempty"`

	// audit logging
	AuditLogging *AuditLoggingSettings `json:"auditLogging,omitempty"`

	// cloud
	Cloud *CloudSpec `json:"cloud,omitempty"`

	// cluster network
	ClusterNetwork *ClusterNetworkingConfig `json:"clusterNetwork,omitempty"`

	// cni plugin
	CniPlugin *CNIPluginSettings `json:"cniPlugin,omitempty"`

	// event rate limit config
	EventRateLimitConfig *EventRateLimitConfig `json:"eventRateLimitConfig,omitempty"`

	// expose strategy
	ExposeStrategy ExposeStrategy `json:"exposeStrategy,omitempty"`

	// kubernetes dashboard
	KubernetesDashboard *KubernetesDashboard `json:"kubernetesDashboard,omitempty"`

	// mla
	Mla *MLASettings `json:"mla,omitempty"`

	// oidc
	Oidc *OIDCSettings `json:"oidc,omitempty"`

	// opa integration
	OpaIntegration *OPAIntegrationSettings `json:"opaIntegration,omitempty"`

	// service account
	ServiceAccount *ServiceAccountSettings `json:"serviceAccount,omitempty"`

	// update window
	UpdateWindow *UpdateWindow `json:"updateWindow,omitempty"`

	// version
	Version Semver `json:"version,omitempty"`
}

// Validate validates this cluster spec
func (m *ClusterSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateMachineNetworks(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateAuditLogging(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateCloud(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateClusterNetwork(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateCniPlugin(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateEventRateLimitConfig(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateExposeStrategy(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateKubernetesDashboard(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMla(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOidc(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateOpaIntegration(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateServiceAccount(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUpdateWindow(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateVersion(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ClusterSpec) validateMachineNetworks(formats strfmt.Registry) error {
	if swag.IsZero(m.MachineNetworks) { // not required
		return nil
	}

	for i := 0; i < len(m.MachineNetworks); i++ {
		if swag.IsZero(m.MachineNetworks[i]) { // not required
			continue
		}

		if m.MachineNetworks[i] != nil {
			if err := m.MachineNetworks[i].Validate(formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("machineNetworks" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("machineNetworks" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *ClusterSpec) validateAuditLogging(formats strfmt.Registry) error {
	if swag.IsZero(m.AuditLogging) { // not required
		return nil
	}

	if m.AuditLogging != nil {
		if err := m.AuditLogging.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("auditLogging")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("auditLogging")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) validateCloud(formats strfmt.Registry) error {
	if swag.IsZero(m.Cloud) { // not required
		return nil
	}

	if m.Cloud != nil {
		if err := m.Cloud.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("cloud")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("cloud")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) validateClusterNetwork(formats strfmt.Registry) error {
	if swag.IsZero(m.ClusterNetwork) { // not required
		return nil
	}

	if m.ClusterNetwork != nil {
		if err := m.ClusterNetwork.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("clusterNetwork")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("clusterNetwork")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) validateCniPlugin(formats strfmt.Registry) error {
	if swag.IsZero(m.CniPlugin) { // not required
		return nil
	}

	if m.CniPlugin != nil {
		if err := m.CniPlugin.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("cniPlugin")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("cniPlugin")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) validateEventRateLimitConfig(formats strfmt.Registry) error {
	if swag.IsZero(m.EventRateLimitConfig) { // not required
		return nil
	}

	if m.EventRateLimitConfig != nil {
		if err := m.EventRateLimitConfig.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("eventRateLimitConfig")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("eventRateLimitConfig")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) validateExposeStrategy(formats strfmt.Registry) error {
	if swag.IsZero(m.ExposeStrategy) { // not required
		return nil
	}

	if err := m.ExposeStrategy.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("exposeStrategy")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("exposeStrategy")
		}
		return err
	}

	return nil
}

func (m *ClusterSpec) validateKubernetesDashboard(formats strfmt.Registry) error {
	if swag.IsZero(m.KubernetesDashboard) { // not required
		return nil
	}

	if m.KubernetesDashboard != nil {
		if err := m.KubernetesDashboard.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kubernetesDashboard")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("kubernetesDashboard")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) validateMla(formats strfmt.Registry) error {
	if swag.IsZero(m.Mla) { // not required
		return nil
	}

	if m.Mla != nil {
		if err := m.Mla.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("mla")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("mla")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) validateOidc(formats strfmt.Registry) error {
	if swag.IsZero(m.Oidc) { // not required
		return nil
	}

	if m.Oidc != nil {
		if err := m.Oidc.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("oidc")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("oidc")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) validateOpaIntegration(formats strfmt.Registry) error {
	if swag.IsZero(m.OpaIntegration) { // not required
		return nil
	}

	if m.OpaIntegration != nil {
		if err := m.OpaIntegration.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("opaIntegration")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("opaIntegration")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) validateServiceAccount(formats strfmt.Registry) error {
	if swag.IsZero(m.ServiceAccount) { // not required
		return nil
	}

	if m.ServiceAccount != nil {
		if err := m.ServiceAccount.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("serviceAccount")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("serviceAccount")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) validateUpdateWindow(formats strfmt.Registry) error {
	if swag.IsZero(m.UpdateWindow) { // not required
		return nil
	}

	if m.UpdateWindow != nil {
		if err := m.UpdateWindow.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("updateWindow")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("updateWindow")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) validateVersion(formats strfmt.Registry) error {
	if swag.IsZero(m.Version) { // not required
		return nil
	}

	if err := m.Version.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("version")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("version")
		}
		return err
	}

	return nil
}

// ContextValidate validate this cluster spec based on the context it is used
func (m *ClusterSpec) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateMachineNetworks(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateAuditLogging(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateCloud(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateClusterNetwork(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateCniPlugin(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateEventRateLimitConfig(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateExposeStrategy(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateKubernetesDashboard(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateMla(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateOidc(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateOpaIntegration(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateServiceAccount(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateUpdateWindow(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateVersion(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ClusterSpec) contextValidateMachineNetworks(ctx context.Context, formats strfmt.Registry) error {

	for i := 0; i < len(m.MachineNetworks); i++ {

		if m.MachineNetworks[i] != nil {
			if err := m.MachineNetworks[i].ContextValidate(ctx, formats); err != nil {
				if ve, ok := err.(*errors.Validation); ok {
					return ve.ValidateName("machineNetworks" + "." + strconv.Itoa(i))
				} else if ce, ok := err.(*errors.CompositeError); ok {
					return ce.ValidateName("machineNetworks" + "." + strconv.Itoa(i))
				}
				return err
			}
		}

	}

	return nil
}

func (m *ClusterSpec) contextValidateAuditLogging(ctx context.Context, formats strfmt.Registry) error {

	if m.AuditLogging != nil {
		if err := m.AuditLogging.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("auditLogging")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("auditLogging")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) contextValidateCloud(ctx context.Context, formats strfmt.Registry) error {

	if m.Cloud != nil {
		if err := m.Cloud.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("cloud")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("cloud")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) contextValidateClusterNetwork(ctx context.Context, formats strfmt.Registry) error {

	if m.ClusterNetwork != nil {
		if err := m.ClusterNetwork.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("clusterNetwork")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("clusterNetwork")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) contextValidateCniPlugin(ctx context.Context, formats strfmt.Registry) error {

	if m.CniPlugin != nil {
		if err := m.CniPlugin.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("cniPlugin")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("cniPlugin")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) contextValidateEventRateLimitConfig(ctx context.Context, formats strfmt.Registry) error {

	if m.EventRateLimitConfig != nil {
		if err := m.EventRateLimitConfig.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("eventRateLimitConfig")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("eventRateLimitConfig")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) contextValidateExposeStrategy(ctx context.Context, formats strfmt.Registry) error {

	if err := m.ExposeStrategy.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("exposeStrategy")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("exposeStrategy")
		}
		return err
	}

	return nil
}

func (m *ClusterSpec) contextValidateKubernetesDashboard(ctx context.Context, formats strfmt.Registry) error {

	if m.KubernetesDashboard != nil {
		if err := m.KubernetesDashboard.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("kubernetesDashboard")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("kubernetesDashboard")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) contextValidateMla(ctx context.Context, formats strfmt.Registry) error {

	if m.Mla != nil {
		if err := m.Mla.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("mla")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("mla")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) contextValidateOidc(ctx context.Context, formats strfmt.Registry) error {

	if m.Oidc != nil {
		if err := m.Oidc.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("oidc")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("oidc")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) contextValidateOpaIntegration(ctx context.Context, formats strfmt.Registry) error {

	if m.OpaIntegration != nil {
		if err := m.OpaIntegration.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("opaIntegration")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("opaIntegration")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) contextValidateServiceAccount(ctx context.Context, formats strfmt.Registry) error {

	if m.ServiceAccount != nil {
		if err := m.ServiceAccount.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("serviceAccount")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("serviceAccount")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) contextValidateUpdateWindow(ctx context.Context, formats strfmt.Registry) error {

	if m.UpdateWindow != nil {
		if err := m.UpdateWindow.ContextValidate(ctx, formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("updateWindow")
			} else if ce, ok := err.(*errors.CompositeError); ok {
				return ce.ValidateName("updateWindow")
			}
			return err
		}
	}

	return nil
}

func (m *ClusterSpec) contextValidateVersion(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Version.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("version")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("version")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ClusterSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ClusterSpec) UnmarshalBinary(b []byte) error {
	var res ClusterSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
