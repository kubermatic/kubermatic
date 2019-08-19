// Code generated by go-swagger; DO NOT EDIT.

package tokens

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"
)

// New creates a new tokens API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) *Client {
	return &Client{transport: transport, formats: formats}
}

/*
Client for tokens API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

/*
AddTokenToServiceAccount Generates a token for the given service account
*/
func (a *Client) AddTokenToServiceAccount(params *AddTokenToServiceAccountParams, authInfo runtime.ClientAuthInfoWriter) (*AddTokenToServiceAccountCreated, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddTokenToServiceAccountParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "addTokenToServiceAccount",
		Method:             "POST",
		PathPattern:        "/api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &AddTokenToServiceAccountReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*AddTokenToServiceAccountCreated), nil

}

/*
DeleteServiceAccountToken Deletes the token
*/
func (a *Client) DeleteServiceAccountToken(params *DeleteServiceAccountTokenParams, authInfo runtime.ClientAuthInfoWriter) (*DeleteServiceAccountTokenOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteServiceAccountTokenParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "deleteServiceAccountToken",
		Method:             "DELETE",
		PathPattern:        "/api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens/{token_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &DeleteServiceAccountTokenReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*DeleteServiceAccountTokenOK), nil

}

/*
ListServiceAccountTokens List tokens for the given service account
*/
func (a *Client) ListServiceAccountTokens(params *ListServiceAccountTokensParams, authInfo runtime.ClientAuthInfoWriter) (*ListServiceAccountTokensOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListServiceAccountTokensParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listServiceAccountTokens",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListServiceAccountTokensReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListServiceAccountTokensOK), nil

}

/*
PatchServiceAccountToken Patches the token name
*/
func (a *Client) PatchServiceAccountToken(params *PatchServiceAccountTokenParams, authInfo runtime.ClientAuthInfoWriter) (*PatchServiceAccountTokenOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPatchServiceAccountTokenParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "patchServiceAccountToken",
		Method:             "PATCH",
		PathPattern:        "/api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens/{token_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &PatchServiceAccountTokenReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*PatchServiceAccountTokenOK), nil

}

/*
UpdateServiceAccountToken Updates and regenerates the token
*/
func (a *Client) UpdateServiceAccountToken(params *UpdateServiceAccountTokenParams, authInfo runtime.ClientAuthInfoWriter) (*UpdateServiceAccountTokenOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewUpdateServiceAccountTokenParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "updateServiceAccountToken",
		Method:             "PUT",
		PathPattern:        "/api/v1/projects/{project_id}/serviceaccounts/{serviceaccount_id}/tokens/{token_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &UpdateServiceAccountTokenReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*UpdateServiceAccountTokenOK), nil

}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
