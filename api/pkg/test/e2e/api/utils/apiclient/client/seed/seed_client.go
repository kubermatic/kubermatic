// Code generated by go-swagger; DO NOT EDIT.

package seed

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new seed API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for seed API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientService is the interface for Client methods
type ClientService interface {
	ListSeedNames(params *ListSeedNamesParams, authInfo runtime.ClientAuthInfoWriter) (*ListSeedNamesOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  ListSeedNames list seed names API
*/
func (a *Client) ListSeedNames(params *ListSeedNamesParams, authInfo runtime.ClientAuthInfoWriter) (*ListSeedNamesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListSeedNamesParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listSeedNames",
		Method:             "GET",
		PathPattern:        "/api/v1/seed",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListSeedNamesReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListSeedNamesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListSeedNamesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
