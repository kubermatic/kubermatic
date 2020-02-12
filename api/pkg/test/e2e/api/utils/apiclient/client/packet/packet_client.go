// Code generated by go-swagger; DO NOT EDIT.

package packet

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new packet API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for packet API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientService is the interface for Client methods
type ClientService interface {
	ListPacketSizes(params *ListPacketSizesParams, authInfo runtime.ClientAuthInfoWriter) (*ListPacketSizesOK, error)

	ListPacketSizesNoCredentials(params *ListPacketSizesNoCredentialsParams, authInfo runtime.ClientAuthInfoWriter) (*ListPacketSizesNoCredentialsOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  ListPacketSizes Lists sizes from packet
*/
func (a *Client) ListPacketSizes(params *ListPacketSizesParams, authInfo runtime.ClientAuthInfoWriter) (*ListPacketSizesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListPacketSizesParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listPacketSizes",
		Method:             "GET",
		PathPattern:        "/api/v1/providers/packet/sizes",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListPacketSizesReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListPacketSizesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListPacketSizesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListPacketSizesNoCredentials Lists sizes from packet
*/
func (a *Client) ListPacketSizesNoCredentials(params *ListPacketSizesNoCredentialsParams, authInfo runtime.ClientAuthInfoWriter) (*ListPacketSizesNoCredentialsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListPacketSizesNoCredentialsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listPacketSizesNoCredentials",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/packet/sizes",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListPacketSizesNoCredentialsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListPacketSizesNoCredentialsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListPacketSizesNoCredentialsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
