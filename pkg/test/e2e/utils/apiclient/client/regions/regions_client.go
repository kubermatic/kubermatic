// Code generated by go-swagger; DO NOT EDIT.

package regions

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new regions API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for regions API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	ListEC2Regions(params *ListEC2RegionsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEC2RegionsOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  ListEC2Regions lists e c2 regions
*/
func (a *Client) ListEC2Regions(params *ListEC2RegionsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEC2RegionsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListEC2RegionsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listEC2Regions",
		Method:             "GET",
		PathPattern:        "/api/v2/providers/ec2/regions",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListEC2RegionsReader{formats: a.formats},
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
	success, ok := result.(*ListEC2RegionsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListEC2RegionsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
