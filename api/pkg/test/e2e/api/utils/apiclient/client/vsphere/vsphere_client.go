// Code generated by go-swagger; DO NOT EDIT.

package vsphere

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"
)

// New creates a new vsphere API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) *Client {
	return &Client{transport: transport, formats: formats}
}

/*
Client for vsphere API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

/*
ListVSphereFolders Lists folders from vsphere datacenter
*/
func (a *Client) ListVSphereFolders(params *ListVSphereFoldersParams, authInfo runtime.ClientAuthInfoWriter) (*ListVSphereFoldersOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListVSphereFoldersParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listVSphereFolders",
		Method:             "GET",
		PathPattern:        "/api/v1/providers/vsphere/folders",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListVSphereFoldersReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListVSphereFoldersOK), nil

}

/*
ListVSphereFoldersNoCredentials Lists folders from vsphere datacenter
*/
func (a *Client) ListVSphereFoldersNoCredentials(params *ListVSphereFoldersNoCredentialsParams, authInfo runtime.ClientAuthInfoWriter) (*ListVSphereFoldersNoCredentialsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListVSphereFoldersNoCredentialsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listVSphereFoldersNoCredentials",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/vsphere/folders",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListVSphereFoldersNoCredentialsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListVSphereFoldersNoCredentialsOK), nil

}

/*
ListVSphereNetworks Lists networks from vsphere datacenter
*/
func (a *Client) ListVSphereNetworks(params *ListVSphereNetworksParams, authInfo runtime.ClientAuthInfoWriter) (*ListVSphereNetworksOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListVSphereNetworksParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listVSphereNetworks",
		Method:             "GET",
		PathPattern:        "/api/v1/providers/vsphere/networks",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListVSphereNetworksReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListVSphereNetworksOK), nil

}

/*
ListVSphereNetworksNoCredentials Lists networks from vsphere datacenter
*/
func (a *Client) ListVSphereNetworksNoCredentials(params *ListVSphereNetworksNoCredentialsParams, authInfo runtime.ClientAuthInfoWriter) (*ListVSphereNetworksNoCredentialsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListVSphereNetworksNoCredentialsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listVSphereNetworksNoCredentials",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/vsphere/networks",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListVSphereNetworksNoCredentialsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListVSphereNetworksNoCredentialsOK), nil

}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
