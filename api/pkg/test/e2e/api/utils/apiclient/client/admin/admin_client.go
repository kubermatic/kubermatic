// Code generated by go-swagger; DO NOT EDIT.

package admin

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"
)

// New creates a new admin API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) *Client {
	return &Client{transport: transport, formats: formats}
}

/*
Client for admin API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

/*
GetAdmins returns list of admin users
*/
func (a *Client) GetAdmins(params *GetAdminsParams, authInfo runtime.ClientAuthInfoWriter) (*GetAdminsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetAdminsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "getAdmins",
		Method:             "GET",
		PathPattern:        "/api/v1/admin",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetAdminsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*GetAdminsOK), nil

}

/*
GetKubermaticSettings gets the global settings
*/
func (a *Client) GetKubermaticSettings(params *GetKubermaticSettingsParams, authInfo runtime.ClientAuthInfoWriter) (*GetKubermaticSettingsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetKubermaticSettingsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "getKubermaticSettings",
		Method:             "GET",
		PathPattern:        "/api/v1/admin/settings",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetKubermaticSettingsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*GetKubermaticSettingsOK), nil

}

/*
PatchKubermaticSettings patches the global settings
*/
func (a *Client) PatchKubermaticSettings(params *PatchKubermaticSettingsParams, authInfo runtime.ClientAuthInfoWriter) (*PatchKubermaticSettingsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPatchKubermaticSettingsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "patchKubermaticSettings",
		Method:             "PATCH",
		PathPattern:        "/api/v1/admin/settings",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &PatchKubermaticSettingsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*PatchKubermaticSettingsOK), nil

}

/*
SetAdmin allows setting and clearing admin role for users
*/
func (a *Client) SetAdmin(params *SetAdminParams, authInfo runtime.ClientAuthInfoWriter) (*SetAdminOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewSetAdminParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "setAdmin",
		Method:             "PUT",
		PathPattern:        "/api/v1/admin",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &SetAdminReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*SetAdminOK), nil

}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
