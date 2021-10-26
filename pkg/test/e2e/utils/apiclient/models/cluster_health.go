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

// ClusterHealth ClusterHealth stores health information about the cluster's components.
//
// swagger:model ClusterHealth
type ClusterHealth struct {

	// apiserver
	Apiserver HealthStatus `json:"apiserver,omitempty"`

	// cloud provider infrastructure
	CloudProviderInfrastructure HealthStatus `json:"cloudProviderInfrastructure,omitempty"`

	// controller
	Controller HealthStatus `json:"controller,omitempty"`

	// etcd
	Etcd HealthStatus `json:"etcd,omitempty"`

	// gatekeeper audit
	GatekeeperAudit HealthStatus `json:"gatekeeperAudit,omitempty"`

	// gatekeeper controller
	GatekeeperController HealthStatus `json:"gatekeeperController,omitempty"`

	// logging
	Logging HealthStatus `json:"logging,omitempty"`

	// machine controller
	MachineController HealthStatus `json:"machineController,omitempty"`

	// monitoring
	Monitoring HealthStatus `json:"monitoring,omitempty"`

	// scheduler
	Scheduler HealthStatus `json:"scheduler,omitempty"`

	// user cluster controller manager
	UserClusterControllerManager HealthStatus `json:"userClusterControllerManager,omitempty"`
}

// Validate validates this cluster health
func (m *ClusterHealth) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateApiserver(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateCloudProviderInfrastructure(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateController(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateEtcd(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateGatekeeperAudit(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateGatekeeperController(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateLogging(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMachineController(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMonitoring(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateScheduler(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateUserClusterControllerManager(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ClusterHealth) validateApiserver(formats strfmt.Registry) error {
	if swag.IsZero(m.Apiserver) { // not required
		return nil
	}

	if err := m.Apiserver.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("apiserver")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) validateCloudProviderInfrastructure(formats strfmt.Registry) error {
	if swag.IsZero(m.CloudProviderInfrastructure) { // not required
		return nil
	}

	if err := m.CloudProviderInfrastructure.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("cloudProviderInfrastructure")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) validateController(formats strfmt.Registry) error {
	if swag.IsZero(m.Controller) { // not required
		return nil
	}

	if err := m.Controller.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("controller")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) validateEtcd(formats strfmt.Registry) error {
	if swag.IsZero(m.Etcd) { // not required
		return nil
	}

	if err := m.Etcd.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("etcd")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) validateGatekeeperAudit(formats strfmt.Registry) error {
	if swag.IsZero(m.GatekeeperAudit) { // not required
		return nil
	}

	if err := m.GatekeeperAudit.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("gatekeeperAudit")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) validateGatekeeperController(formats strfmt.Registry) error {
	if swag.IsZero(m.GatekeeperController) { // not required
		return nil
	}

	if err := m.GatekeeperController.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("gatekeeperController")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) validateLogging(formats strfmt.Registry) error {
	if swag.IsZero(m.Logging) { // not required
		return nil
	}

	if err := m.Logging.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("logging")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) validateMachineController(formats strfmt.Registry) error {
	if swag.IsZero(m.MachineController) { // not required
		return nil
	}

	if err := m.MachineController.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("machineController")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) validateMonitoring(formats strfmt.Registry) error {
	if swag.IsZero(m.Monitoring) { // not required
		return nil
	}

	if err := m.Monitoring.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("monitoring")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) validateScheduler(formats strfmt.Registry) error {
	if swag.IsZero(m.Scheduler) { // not required
		return nil
	}

	if err := m.Scheduler.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("scheduler")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) validateUserClusterControllerManager(formats strfmt.Registry) error {
	if swag.IsZero(m.UserClusterControllerManager) { // not required
		return nil
	}

	if err := m.UserClusterControllerManager.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("userClusterControllerManager")
		}
		return err
	}

	return nil
}

// ContextValidate validate this cluster health based on the context it is used
func (m *ClusterHealth) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateApiserver(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateCloudProviderInfrastructure(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateController(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateEtcd(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateGatekeeperAudit(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateGatekeeperController(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateLogging(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateMachineController(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateMonitoring(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateScheduler(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateUserClusterControllerManager(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ClusterHealth) contextValidateApiserver(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Apiserver.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("apiserver")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) contextValidateCloudProviderInfrastructure(ctx context.Context, formats strfmt.Registry) error {

	if err := m.CloudProviderInfrastructure.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("cloudProviderInfrastructure")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) contextValidateController(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Controller.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("controller")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) contextValidateEtcd(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Etcd.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("etcd")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) contextValidateGatekeeperAudit(ctx context.Context, formats strfmt.Registry) error {

	if err := m.GatekeeperAudit.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("gatekeeperAudit")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) contextValidateGatekeeperController(ctx context.Context, formats strfmt.Registry) error {

	if err := m.GatekeeperController.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("gatekeeperController")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) contextValidateLogging(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Logging.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("logging")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) contextValidateMachineController(ctx context.Context, formats strfmt.Registry) error {

	if err := m.MachineController.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("machineController")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) contextValidateMonitoring(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Monitoring.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("monitoring")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) contextValidateScheduler(ctx context.Context, formats strfmt.Registry) error {

	if err := m.Scheduler.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("scheduler")
		}
		return err
	}

	return nil
}

func (m *ClusterHealth) contextValidateUserClusterControllerManager(ctx context.Context, formats strfmt.Registry) error {

	if err := m.UserClusterControllerManager.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("userClusterControllerManager")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ClusterHealth) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ClusterHealth) UnmarshalBinary(b []byte) error {
	var res ClusterHealth
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
