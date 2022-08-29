// Code generated by go-swagger; DO NOT EDIT.

package users

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new users API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for users API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	AddUserToProject(params *AddUserToProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*AddUserToProjectCreated, error)

	DeleteUserFromProject(params *DeleteUserFromProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteUserFromProjectOK, error)

	EditUserInProject(params *EditUserInProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*EditUserInProjectOK, error)

	GetCurrentUser(params *GetCurrentUserParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetCurrentUserOK, error)

	GetUsersForProject(params *GetUsersForProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetUsersForProjectOK, error)

	LogoutCurrentUser(params *LogoutCurrentUserParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*LogoutCurrentUserOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
AddUserToProject Adds the given user to the given project
*/
func (a *Client) AddUserToProject(params *AddUserToProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*AddUserToProjectCreated, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddUserToProjectParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "addUserToProject",
		Method:             "POST",
		PathPattern:        "/api/v1/projects/{project_id}/users",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &AddUserToProjectReader{formats: a.formats},
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
	success, ok := result.(*AddUserToProjectCreated)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*AddUserToProjectDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
DeleteUserFromProject Removes the given member from the project
*/
func (a *Client) DeleteUserFromProject(params *DeleteUserFromProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteUserFromProjectOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteUserFromProjectParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "deleteUserFromProject",
		Method:             "DELETE",
		PathPattern:        "/api/v1/projects/{project_id}/users/{user_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &DeleteUserFromProjectReader{formats: a.formats},
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
	success, ok := result.(*DeleteUserFromProjectOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*DeleteUserFromProjectDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
EditUserInProject Changes membership of the given user for the given project
*/
func (a *Client) EditUserInProject(params *EditUserInProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*EditUserInProjectOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewEditUserInProjectParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "editUserInProject",
		Method:             "PUT",
		PathPattern:        "/api/v1/projects/{project_id}/users/{user_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &EditUserInProjectReader{formats: a.formats},
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
	success, ok := result.(*EditUserInProjectOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*EditUserInProjectDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetCurrentUser returns information about the current user
*/
func (a *Client) GetCurrentUser(params *GetCurrentUserParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetCurrentUserOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetCurrentUserParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "getCurrentUser",
		Method:             "GET",
		PathPattern:        "/api/v1/me",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetCurrentUserReader{formats: a.formats},
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
	success, ok := result.(*GetCurrentUserOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetCurrentUserDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
GetUsersForProject Get list of users for the given project
*/
func (a *Client) GetUsersForProject(params *GetUsersForProjectParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetUsersForProjectOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetUsersForProjectParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "getUsersForProject",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/users",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetUsersForProjectReader{formats: a.formats},
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
	success, ok := result.(*GetUsersForProjectOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetUsersForProjectDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
LogoutCurrentUser adds current authorization bearer token to the blacklist

Enforces user to login again with the new token.
*/
func (a *Client) LogoutCurrentUser(params *LogoutCurrentUserParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*LogoutCurrentUserOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewLogoutCurrentUserParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "logoutCurrentUser",
		Method:             "POST",
		PathPattern:        "/api/v1/me/logout",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &LogoutCurrentUserReader{formats: a.formats},
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
	success, ok := result.(*LogoutCurrentUserOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*LogoutCurrentUserDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
