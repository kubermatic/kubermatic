// Code generated by go-swagger; DO NOT EDIT.

package addon

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"
)

// New creates a new addon API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) *Client {
	return &Client{transport: transport, formats: formats}
}

/*
Client for addon API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

/*
CreateAddon Creates an addon that will belong to the given cluster
*/
func (a *Client) CreateAddon(params *CreateAddonParams, authInfo runtime.ClientAuthInfoWriter) (*CreateAddonCreated, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreateAddonParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "createAddon",
		Method:             "POST",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/addons",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &CreateAddonReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*CreateAddonCreated), nil

}

/*
DeleteAddon deletes the given addon that belongs to the cluster
*/
func (a *Client) DeleteAddon(params *DeleteAddonParams, authInfo runtime.ClientAuthInfoWriter) (*DeleteAddonOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteAddonParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "deleteAddon",
		Method:             "DELETE",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/addons/{addon_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &DeleteAddonReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*DeleteAddonOK), nil

}

/*
GetAddon gets an addon that is assigned to the given cluster
*/
func (a *Client) GetAddon(params *GetAddonParams, authInfo runtime.ClientAuthInfoWriter) (*GetAddonOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetAddonParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "getAddon",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/addons/{addon_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetAddonReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*GetAddonOK), nil

}

/*
ListAddons Lists addons that belong to the given cluster
*/
func (a *Client) ListAddons(params *ListAddonsParams, authInfo runtime.ClientAuthInfoWriter) (*ListAddonsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAddonsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listAddons",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/addons",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAddonsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListAddonsOK), nil

}

/*
PatchAddon patches an addon that is assigned to the given cluster
*/
func (a *Client) PatchAddon(params *PatchAddonParams, authInfo runtime.ClientAuthInfoWriter) (*PatchAddonOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPatchAddonParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "patchAddon",
		Method:             "PATCH",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/addons/{addon_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &PatchAddonReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*PatchAddonOK), nil

}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
