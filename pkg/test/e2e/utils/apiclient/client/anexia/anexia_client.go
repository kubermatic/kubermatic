// Code generated by go-swagger; DO NOT EDIT.

package anexia

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new anexia API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for anexia API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientService is the interface for Client methods
type ClientService interface {
	ListAnexiaTemplates(params *ListAnexiaTemplatesParams, authInfo runtime.ClientAuthInfoWriter) (*ListAnexiaTemplatesOK, error)

	ListAnexiaTemplatesNoCredentialsV2(params *ListAnexiaTemplatesNoCredentialsV2Params, authInfo runtime.ClientAuthInfoWriter) (*ListAnexiaTemplatesNoCredentialsV2OK, error)

	ListAnexiaVlans(params *ListAnexiaVlansParams, authInfo runtime.ClientAuthInfoWriter) (*ListAnexiaVlansOK, error)

	ListAnexiaVlansNoCredentialsV2(params *ListAnexiaVlansNoCredentialsV2Params, authInfo runtime.ClientAuthInfoWriter) (*ListAnexiaVlansNoCredentialsV2OK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  ListAnexiaTemplates Lists templates from anexia
*/
func (a *Client) ListAnexiaTemplates(params *ListAnexiaTemplatesParams, authInfo runtime.ClientAuthInfoWriter) (*ListAnexiaTemplatesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAnexiaTemplatesParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listAnexiaTemplates",
		Method:             "GET",
		PathPattern:        "/api/v1/providers/anexia/templates",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAnexiaTemplatesReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListAnexiaTemplatesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAnexiaTemplatesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListAnexiaTemplatesNoCredentialsV2 Lists templates from Anexia
*/
func (a *Client) ListAnexiaTemplatesNoCredentialsV2(params *ListAnexiaTemplatesNoCredentialsV2Params, authInfo runtime.ClientAuthInfoWriter) (*ListAnexiaTemplatesNoCredentialsV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAnexiaTemplatesNoCredentialsV2Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listAnexiaTemplatesNoCredentialsV2",
		Method:             "GET",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/providers/anexia/templates",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAnexiaTemplatesNoCredentialsV2Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListAnexiaTemplatesNoCredentialsV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAnexiaTemplatesNoCredentialsV2Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListAnexiaVlans Lists vlans from anexia
*/
func (a *Client) ListAnexiaVlans(params *ListAnexiaVlansParams, authInfo runtime.ClientAuthInfoWriter) (*ListAnexiaVlansOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAnexiaVlansParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listAnexiaVlans",
		Method:             "GET",
		PathPattern:        "/api/v1/providers/anexia/vlans",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAnexiaVlansReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListAnexiaVlansOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAnexiaVlansDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListAnexiaVlansNoCredentialsV2 Lists vlans from Anexia
*/
func (a *Client) ListAnexiaVlansNoCredentialsV2(params *ListAnexiaVlansNoCredentialsV2Params, authInfo runtime.ClientAuthInfoWriter) (*ListAnexiaVlansNoCredentialsV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAnexiaVlansNoCredentialsV2Params()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listAnexiaVlansNoCredentialsV2",
		Method:             "GET",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/providers/anexia/vlans",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAnexiaVlansNoCredentialsV2Reader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListAnexiaVlansNoCredentialsV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAnexiaVlansNoCredentialsV2Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
