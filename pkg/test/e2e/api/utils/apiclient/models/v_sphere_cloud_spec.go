// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// VSphereCloudSpec VSphereCloudSpec specifies access data to VSphere cloud.
//
// swagger:model VSphereCloudSpec
type VSphereCloudSpec struct {

	// Datastore to be used for storing virtual machines, it is mutually
	// exclusive with DatastoreCluster.
	// +optional
	Datastore string `json:"datastore,omitempty"`

	// DatastoreCluster to be used for determining the Datastore for virtual
	// machines, it is mutually exclusive with Datastore.
	// +optional
	DatastoreCluster string `json:"datastoreCluster,omitempty"`

	// Folder is the folder to be used to group the provisioned virtual
	// machines.
	// +optional
	Folder string `json:"folder,omitempty"`

	// Password is the vSphere user password.
	// +optional
	Password string `json:"password,omitempty"`

	// Username is the vSphere user name.
	// +optional
	Username string `json:"username,omitempty"`

	// VMNetName is the name of the vSphere network.
	VMNetName string `json:"vmNetName,omitempty"`

	// credentials reference
	CredentialsReference GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// infra management user
	InfraManagementUser *VSphereCredentials `json:"infraManagementUser,omitempty"`
}

// Validate validates this v sphere cloud spec
func (m *VSphereCloudSpec) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateCredentialsReference(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateInfraManagementUser(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *VSphereCloudSpec) validateCredentialsReference(formats strfmt.Registry) error {

	if swag.IsZero(m.CredentialsReference) { // not required
		return nil
	}

	if err := m.CredentialsReference.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("credentialsReference")
		}
		return err
	}

	return nil
}

func (m *VSphereCloudSpec) validateInfraManagementUser(formats strfmt.Registry) error {

	if swag.IsZero(m.InfraManagementUser) { // not required
		return nil
	}

	if m.InfraManagementUser != nil {
		if err := m.InfraManagementUser.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("infraManagementUser")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *VSphereCloudSpec) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *VSphereCloudSpec) UnmarshalBinary(b []byte) error {
	var res VSphereCloudSpec
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
