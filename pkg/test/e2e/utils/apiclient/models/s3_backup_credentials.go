// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// S3BackupCredentials S3BackupCredentials contains credentials for S3 etcd backups
//
// swagger:model S3BackupCredentials
type S3BackupCredentials struct {

	// access key ID
	AccessKeyID string `json:"accessKeyId,omitempty"`

	// secret access key
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
}

// Validate validates this s3 backup credentials
func (m *S3BackupCredentials) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this s3 backup credentials based on context it is used
func (m *S3BackupCredentials) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *S3BackupCredentials) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *S3BackupCredentials) UnmarshalBinary(b []byte) error {
	var res S3BackupCredentials
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
