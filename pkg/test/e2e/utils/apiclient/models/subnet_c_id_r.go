// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
)

// SubnetCIDR +kubebuilder:validation:Pattern="((^((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))/([0-9]|[1-2][0-9]|3[0-2])$)|(^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:))/([0-9]|[0-9][0-9]|1[0-1][0-9]|12[0-8])$))"
//
// SubnetCIDR is used to store IPv4/IPv6 CIDR.
//
// swagger:model SubnetCIDR
type SubnetCIDR string

// Validate validates this subnet c ID r
func (m SubnetCIDR) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this subnet c ID r based on context it is used
func (m SubnetCIDR) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
