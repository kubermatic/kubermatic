// Code generated by go-swagger; DO NOT EDIT.

package eks

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new eks API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for eks API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	ListEKSRegions(params *ListEKSRegionsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEKSRegionsOK, error)

	ListEKSSecurityGroupIDs(params *ListEKSSecurityGroupIDsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEKSSecurityGroupIDsOK, error)

	ListEKSSubnetIDs(params *ListEKSSubnetIDsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEKSSubnetIDsOK, error)

	ListEKSVpcIds(params *ListEKSVpcIdsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEKSVpcIdsOK, error)

	ValidateEKSCredentials(params *ValidateEKSCredentialsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ValidateEKSCredentialsOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  ListEKSRegions lists e k s regions
*/
func (a *Client) ListEKSRegions(params *ListEKSRegionsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEKSRegionsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListEKSRegionsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listEKSRegions",
		Method:             "GET",
		PathPattern:        "/api/v2/providers/eks/regions",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListEKSRegionsReader{formats: a.formats},
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
	success, ok := result.(*ListEKSRegionsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListEKSRegionsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListEKSSecurityGroupIDs lists e k s regions
*/
func (a *Client) ListEKSSecurityGroupIDs(params *ListEKSSecurityGroupIDsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEKSSecurityGroupIDsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListEKSSecurityGroupIDsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listEKSSecurityGroupIDs",
		Method:             "GET",
		PathPattern:        "/api/v2/providers/eks/securityGroupIDs",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListEKSSecurityGroupIDsReader{formats: a.formats},
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
	success, ok := result.(*ListEKSSecurityGroupIDsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListEKSSecurityGroupIDsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListEKSSubnetIDs Lists EKS subnetID list
*/
func (a *Client) ListEKSSubnetIDs(params *ListEKSSubnetIDsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEKSSubnetIDsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListEKSSubnetIDsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listEKSSubnetIDs",
		Method:             "GET",
		PathPattern:        "/api/v2/providers/eks/subnetIDs",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListEKSSubnetIDsReader{formats: a.formats},
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
	success, ok := result.(*ListEKSSubnetIDsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListEKSSubnetIDsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListEKSVpcIds Lists EKS vpcID list
*/
func (a *Client) ListEKSVpcIds(params *ListEKSVpcIdsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEKSVpcIdsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListEKSVpcIdsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listEKSVpcIds",
		Method:             "GET",
		PathPattern:        "/api/v2/providers/eks/vpcIDs",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListEKSVpcIdsReader{formats: a.formats},
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
	success, ok := result.(*ListEKSVpcIdsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListEKSVpcIdsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ValidateEKSCredentials Validates EKS credentials
*/
func (a *Client) ValidateEKSCredentials(params *ValidateEKSCredentialsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ValidateEKSCredentialsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewValidateEKSCredentialsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "validateEKSCredentials",
		Method:             "GET",
		PathPattern:        "/api/v2/providers/eks/validatecredentials",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ValidateEKSCredentialsReader{formats: a.formats},
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
	success, ok := result.(*ValidateEKSCredentialsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ValidateEKSCredentialsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
