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

// ProxySettings ProxySettings allow configuring a HTTP proxy for the controlplanes
// and nodes.
//
// swagger:model ProxySettings
type ProxySettings struct {

	// http proxy
	HTTPProxy ProxyValue `json:"httpProxy,omitempty"`

	// no proxy
	NoProxy ProxyValue `json:"noProxy,omitempty"`
}

// Validate validates this proxy settings
func (m *ProxySettings) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateHTTPProxy(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateNoProxy(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ProxySettings) validateHTTPProxy(formats strfmt.Registry) error {
	if swag.IsZero(m.HTTPProxy) { // not required
		return nil
	}

	if err := m.HTTPProxy.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("httpProxy")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("httpProxy")
		}
		return err
	}

	return nil
}

func (m *ProxySettings) validateNoProxy(formats strfmt.Registry) error {
	if swag.IsZero(m.NoProxy) { // not required
		return nil
	}

	if err := m.NoProxy.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("noProxy")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("noProxy")
		}
		return err
	}

	return nil
}

// ContextValidate validate this proxy settings based on the context it is used
func (m *ProxySettings) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := m.contextValidateHTTPProxy(ctx, formats); err != nil {
		res = append(res, err)
	}

	if err := m.contextValidateNoProxy(ctx, formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ProxySettings) contextValidateHTTPProxy(ctx context.Context, formats strfmt.Registry) error {

	if err := m.HTTPProxy.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("httpProxy")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("httpProxy")
		}
		return err
	}

	return nil
}

func (m *ProxySettings) contextValidateNoProxy(ctx context.Context, formats strfmt.Registry) error {

	if err := m.NoProxy.ContextValidate(ctx, formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("noProxy")
		} else if ce, ok := err.(*errors.CompositeError); ok {
			return ce.ValidateName("noProxy")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ProxySettings) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ProxySettings) UnmarshalBinary(b []byte) error {
	var res ProxySettings
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
