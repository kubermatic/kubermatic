// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// AKSVMSize AKSVMSize is the object representing Azure VM sizes.
//
// swagger:model AKSVMSize
type AKSVMSize struct {

	// max data disk count
	MaxDataDiskCount int32 `json:"maxDataDiskCount,omitempty"`

	// memory in m b
	MemoryInMB int32 `json:"memoryInMB,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// number of cores
	NumberOfCores int32 `json:"numberOfCores,omitempty"`

	// number of g p us
	NumberOfGPUs int32 `json:"numberOfGPUs,omitempty"`

	// os disk size in m b
	OsDiskSizeInMB int32 `json:"osDiskSizeInMB,omitempty"`

	// resource disk size in m b
	ResourceDiskSizeInMB int32 `json:"resourceDiskSizeInMB,omitempty"`
}

// Validate validates this a k s VM size
func (m *AKSVMSize) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this a k s VM size based on context it is used
func (m *AKSVMSize) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *AKSVMSize) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *AKSVMSize) UnmarshalBinary(b []byte) error {
	var res AKSVMSize
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
