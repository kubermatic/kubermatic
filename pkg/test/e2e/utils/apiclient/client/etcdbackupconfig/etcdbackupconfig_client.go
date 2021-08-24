// Code generated by go-swagger; DO NOT EDIT.

package etcdbackupconfig

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new etcdbackupconfig API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for etcdbackupconfig API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	CreateEtcdBackupConfig(params *CreateEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateEtcdBackupConfigCreated, error)

	DeleteEtcdBackupConfig(params *DeleteEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteEtcdBackupConfigOK, error)

	GetEtcdBackupConfig(params *GetEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetEtcdBackupConfigOK, error)

	ListEtcdBackupConfig(params *ListEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEtcdBackupConfigOK, error)

	ListProjectEtcdBackupConfig(params *ListProjectEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListProjectEtcdBackupConfigOK, error)

	PatchEtcdBackupConfig(params *PatchEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PatchEtcdBackupConfigOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  CreateEtcdBackupConfig Creates a etcd backup config that will belong to the given cluster
*/
func (a *Client) CreateEtcdBackupConfig(params *CreateEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateEtcdBackupConfigCreated, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreateEtcdBackupConfigParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "createEtcdBackupConfig",
		Method:             "POST",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &CreateEtcdBackupConfigReader{formats: a.formats},
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
	success, ok := result.(*CreateEtcdBackupConfigCreated)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*CreateEtcdBackupConfigDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  DeleteEtcdBackupConfig Deletes a etcd backup config for a given cluster based on its name
*/
func (a *Client) DeleteEtcdBackupConfig(params *DeleteEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteEtcdBackupConfigOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteEtcdBackupConfigParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "deleteEtcdBackupConfig",
		Method:             "DELETE",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs/{ebc_name}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &DeleteEtcdBackupConfigReader{formats: a.formats},
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
	success, ok := result.(*DeleteEtcdBackupConfigOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*DeleteEtcdBackupConfigDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  GetEtcdBackupConfig Gets a etcd backup config for a given cluster based on its name
*/
func (a *Client) GetEtcdBackupConfig(params *GetEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetEtcdBackupConfigOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetEtcdBackupConfigParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "getEtcdBackupConfig",
		Method:             "GET",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs/{ebc_name}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetEtcdBackupConfigReader{formats: a.formats},
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
	success, ok := result.(*GetEtcdBackupConfigOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetEtcdBackupConfigDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListEtcdBackupConfig List etcd backup configs for a given cluster
*/
func (a *Client) ListEtcdBackupConfig(params *ListEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListEtcdBackupConfigOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListEtcdBackupConfigParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listEtcdBackupConfig",
		Method:             "GET",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListEtcdBackupConfigReader{formats: a.formats},
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
	success, ok := result.(*ListEtcdBackupConfigOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListEtcdBackupConfigDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListProjectEtcdBackupConfig List etcd backup configs for a given project
*/
func (a *Client) ListProjectEtcdBackupConfig(params *ListProjectEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListProjectEtcdBackupConfigOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListProjectEtcdBackupConfigParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listProjectEtcdBackupConfig",
		Method:             "GET",
		PathPattern:        "/api/v2/projects/{project_id}/etcdbackupconfigs",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListProjectEtcdBackupConfigReader{formats: a.formats},
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
	success, ok := result.(*ListProjectEtcdBackupConfigOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListProjectEtcdBackupConfigDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  PatchEtcdBackupConfig Patches a etcd backup config for a given cluster based on its name
*/
func (a *Client) PatchEtcdBackupConfig(params *PatchEtcdBackupConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PatchEtcdBackupConfigOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPatchEtcdBackupConfigParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "patchEtcdBackupConfig",
		Method:             "PATCH",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/etcdbackupconfigs/{ebc_name}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &PatchEtcdBackupConfigReader{formats: a.formats},
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
	success, ok := result.(*PatchEtcdBackupConfigOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PatchEtcdBackupConfigDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
