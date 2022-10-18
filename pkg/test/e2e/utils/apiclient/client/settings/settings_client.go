// Code generated by go-swagger; DO NOT EDIT.

package settings

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new settings API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for settings API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	GetCurrentUserSettings(params *GetCurrentUserSettingsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetCurrentUserSettingsOK, error)

	PatchCurrentUserSettings(params *PatchCurrentUserSettingsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PatchCurrentUserSettingsOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
GetCurrentUserSettings returns settings of the current user
*/
func (a *Client) GetCurrentUserSettings(params *GetCurrentUserSettingsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetCurrentUserSettingsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetCurrentUserSettingsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "getCurrentUserSettings",
		Method:             "GET",
		PathPattern:        "/api/v1/me/settings",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetCurrentUserSettingsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetCurrentUserSettingsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetCurrentUserSettingsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
PatchCurrentUserSettings updates settings of the current user
*/
func (a *Client) PatchCurrentUserSettings(params *PatchCurrentUserSettingsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PatchCurrentUserSettingsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPatchCurrentUserSettingsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "patchCurrentUserSettings",
		Method:             "PATCH",
		PathPattern:        "/api/v1/me/settings",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &PatchCurrentUserSettingsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PatchCurrentUserSettingsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PatchCurrentUserSettingsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
