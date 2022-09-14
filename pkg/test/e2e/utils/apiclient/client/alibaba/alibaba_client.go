// Code generated by go-swagger; DO NOT EDIT.

package alibaba

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new alibaba API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for alibaba API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	ListAlibabaInstanceTypes(params *ListAlibabaInstanceTypesParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaInstanceTypesOK, error)

	ListAlibabaInstanceTypesNoCredentials(params *ListAlibabaInstanceTypesNoCredentialsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaInstanceTypesNoCredentialsOK, error)

	ListAlibabaInstanceTypesNoCredentialsV2(params *ListAlibabaInstanceTypesNoCredentialsV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaInstanceTypesNoCredentialsV2OK, error)

	ListAlibabaVSwitches(params *ListAlibabaVSwitchesParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaVSwitchesOK, error)

	ListAlibabaVSwitchesNoCredentialsV2(params *ListAlibabaVSwitchesNoCredentialsV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaVSwitchesNoCredentialsV2OK, error)

	ListAlibabaZones(params *ListAlibabaZonesParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaZonesOK, error)

	ListAlibabaZonesNoCredentials(params *ListAlibabaZonesNoCredentialsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaZonesNoCredentialsOK, error)

	ListAlibabaZonesNoCredentialsV2(params *ListAlibabaZonesNoCredentialsV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaZonesNoCredentialsV2OK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
ListAlibabaInstanceTypes lists available alibaba instance types
*/
func (a *Client) ListAlibabaInstanceTypes(params *ListAlibabaInstanceTypesParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaInstanceTypesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAlibabaInstanceTypesParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listAlibabaInstanceTypes",
		Method:             "GET",
		PathPattern:        "/api/v1/providers/alibaba/instancetypes",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAlibabaInstanceTypesReader{formats: a.formats},
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
	success, ok := result.(*ListAlibabaInstanceTypesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAlibabaInstanceTypesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ListAlibabaInstanceTypesNoCredentials Lists available Alibaba Instance Types
*/
func (a *Client) ListAlibabaInstanceTypesNoCredentials(params *ListAlibabaInstanceTypesNoCredentialsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaInstanceTypesNoCredentialsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAlibabaInstanceTypesNoCredentialsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listAlibabaInstanceTypesNoCredentials",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/alibaba/instancetypes",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAlibabaInstanceTypesNoCredentialsReader{formats: a.formats},
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
	success, ok := result.(*ListAlibabaInstanceTypesNoCredentialsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAlibabaInstanceTypesNoCredentialsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ListAlibabaInstanceTypesNoCredentialsV2 Lists available Alibaba Instance Types
*/
func (a *Client) ListAlibabaInstanceTypesNoCredentialsV2(params *ListAlibabaInstanceTypesNoCredentialsV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaInstanceTypesNoCredentialsV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAlibabaInstanceTypesNoCredentialsV2Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "listAlibabaInstanceTypesNoCredentialsV2",
		Method:             "GET",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/providers/alibaba/instancetypes",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAlibabaInstanceTypesNoCredentialsV2Reader{formats: a.formats},
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
	success, ok := result.(*ListAlibabaInstanceTypesNoCredentialsV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAlibabaInstanceTypesNoCredentialsV2Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ListAlibabaVSwitches lists available alibaba v switches
*/
func (a *Client) ListAlibabaVSwitches(params *ListAlibabaVSwitchesParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaVSwitchesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAlibabaVSwitchesParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listAlibabaVSwitches",
		Method:             "GET",
		PathPattern:        "/api/v1/providers/alibaba/vswitches",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAlibabaVSwitchesReader{formats: a.formats},
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
	success, ok := result.(*ListAlibabaVSwitchesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAlibabaVSwitchesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ListAlibabaVSwitchesNoCredentialsV2 Lists available Alibaba vSwitches
*/
func (a *Client) ListAlibabaVSwitchesNoCredentialsV2(params *ListAlibabaVSwitchesNoCredentialsV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaVSwitchesNoCredentialsV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAlibabaVSwitchesNoCredentialsV2Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "listAlibabaVSwitchesNoCredentialsV2",
		Method:             "GET",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/providers/alibaba/vswitches",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAlibabaVSwitchesNoCredentialsV2Reader{formats: a.formats},
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
	success, ok := result.(*ListAlibabaVSwitchesNoCredentialsV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAlibabaVSwitchesNoCredentialsV2Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ListAlibabaZones lists available alibaba zones
*/
func (a *Client) ListAlibabaZones(params *ListAlibabaZonesParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaZonesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAlibabaZonesParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listAlibabaZones",
		Method:             "GET",
		PathPattern:        "/api/v1/providers/alibaba/zones",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAlibabaZonesReader{formats: a.formats},
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
	success, ok := result.(*ListAlibabaZonesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAlibabaZonesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ListAlibabaZonesNoCredentials Lists available Alibaba Instance Types
*/
func (a *Client) ListAlibabaZonesNoCredentials(params *ListAlibabaZonesNoCredentialsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaZonesNoCredentialsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAlibabaZonesNoCredentialsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listAlibabaZonesNoCredentials",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/alibaba/zones",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAlibabaZonesNoCredentialsReader{formats: a.formats},
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
	success, ok := result.(*ListAlibabaZonesNoCredentialsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAlibabaZonesNoCredentialsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
ListAlibabaZonesNoCredentialsV2 Lists available Alibaba Instance Types
*/
func (a *Client) ListAlibabaZonesNoCredentialsV2(params *ListAlibabaZonesNoCredentialsV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAlibabaZonesNoCredentialsV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAlibabaZonesNoCredentialsV2Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "listAlibabaZonesNoCredentialsV2",
		Method:             "GET",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/providers/alibaba/zones",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAlibabaZonesNoCredentialsV2Reader{formats: a.formats},
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
	success, ok := result.(*ListAlibabaZonesNoCredentialsV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAlibabaZonesNoCredentialsV2Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
