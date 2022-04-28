// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
)

// ExposeStrategy ExposeStrategy is the strategy used to expose a cluster control plane.
//
// Possible values are `NodePort`, `LoadBalancer` or `Tunneling` (requires a feature gate).
//
// swagger:model ExposeStrategy
type ExposeStrategy string

// Validate validates this expose strategy
func (m ExposeStrategy) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this expose strategy based on context it is used
func (m ExposeStrategy) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}
