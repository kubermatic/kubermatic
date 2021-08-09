// Code generated by go-swagger; DO NOT EDIT.

package addon

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new addon API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for addon API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	CreateAddon(params *CreateAddonParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateAddonCreated, error)

	CreateAddonV2(params *CreateAddonV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateAddonV2Created, error)

	DeleteAddon(params *DeleteAddonParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteAddonOK, error)

	DeleteAddonV2(params *DeleteAddonV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteAddonV2OK, error)

	GetAddon(params *GetAddonParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetAddonOK, error)

	GetAddonV2(params *GetAddonV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetAddonV2OK, error)

	ListAccessibleAddons(params *ListAccessibleAddonsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAccessibleAddonsOK, error)

	ListAddons(params *ListAddonsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAddonsOK, error)

	ListAddonsV2(params *ListAddonsV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAddonsV2OK, error)

	ListInstallableAddons(params *ListInstallableAddonsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListInstallableAddonsOK, error)

	ListInstallableAddonsV2(params *ListInstallableAddonsV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListInstallableAddonsV2OK, error)

	PatchAddon(params *PatchAddonParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PatchAddonOK, error)

	PatchAddonV2(params *PatchAddonV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PatchAddonV2OK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  CreateAddon Creates an addon that will belong to the given cluster
*/
func (a *Client) CreateAddon(params *CreateAddonParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateAddonCreated, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreateAddonParams()
	}
	op := &runtime.ClientOperation{
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
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*CreateAddonCreated)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*CreateAddonDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  CreateAddonV2 Creates an addon that will belong to the given cluster
*/
func (a *Client) CreateAddonV2(params *CreateAddonV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*CreateAddonV2Created, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreateAddonV2Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "createAddonV2",
		Method:             "POST",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/addons",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &CreateAddonV2Reader{formats: a.formats},
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
	success, ok := result.(*CreateAddonV2Created)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*CreateAddonV2Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  DeleteAddon deletes the given addon that belongs to the cluster
*/
func (a *Client) DeleteAddon(params *DeleteAddonParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteAddonOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteAddonParams()
	}
	op := &runtime.ClientOperation{
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
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*DeleteAddonOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*DeleteAddonDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  DeleteAddonV2 deletes the given addon that belongs to the cluster
*/
func (a *Client) DeleteAddonV2(params *DeleteAddonV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*DeleteAddonV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteAddonV2Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "deleteAddonV2",
		Method:             "DELETE",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &DeleteAddonV2Reader{formats: a.formats},
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
	success, ok := result.(*DeleteAddonV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*DeleteAddonV2Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  GetAddon gets an addon that is assigned to the given cluster
*/
func (a *Client) GetAddon(params *GetAddonParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetAddonOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetAddonParams()
	}
	op := &runtime.ClientOperation{
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
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetAddonOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetAddonDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  GetAddonV2 gets an addon that is assigned to the given cluster
*/
func (a *Client) GetAddonV2(params *GetAddonV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*GetAddonV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetAddonV2Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "getAddonV2",
		Method:             "GET",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetAddonV2Reader{formats: a.formats},
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
	success, ok := result.(*GetAddonV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetAddonV2Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListAccessibleAddons Lists names of addons that can be configured inside the user clusters
*/
func (a *Client) ListAccessibleAddons(params *ListAccessibleAddonsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAccessibleAddonsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAccessibleAddonsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listAccessibleAddons",
		Method:             "POST",
		PathPattern:        "/api/v1/addons",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAccessibleAddonsReader{formats: a.formats},
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
	success, ok := result.(*ListAccessibleAddonsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAccessibleAddonsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListAddons Lists addons that belong to the given cluster
*/
func (a *Client) ListAddons(params *ListAddonsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAddonsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAddonsParams()
	}
	op := &runtime.ClientOperation{
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
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListAddonsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAddonsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListAddonsV2 Lists addons that belong to the given cluster
*/
func (a *Client) ListAddonsV2(params *ListAddonsV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListAddonsV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListAddonsV2Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "listAddonsV2",
		Method:             "GET",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/addons",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListAddonsV2Reader{formats: a.formats},
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
	success, ok := result.(*ListAddonsV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListAddonsV2Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListInstallableAddons Lists names of addons that can be installed inside the user cluster
*/
func (a *Client) ListInstallableAddons(params *ListInstallableAddonsParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListInstallableAddonsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListInstallableAddonsParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "listInstallableAddons",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/installableaddons",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListInstallableAddonsReader{formats: a.formats},
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
	success, ok := result.(*ListInstallableAddonsOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListInstallableAddonsDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListInstallableAddonsV2 Lists names of addons that can be installed inside the user cluster
*/
func (a *Client) ListInstallableAddonsV2(params *ListInstallableAddonsV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*ListInstallableAddonsV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListInstallableAddonsV2Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "listInstallableAddonsV2",
		Method:             "GET",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/installableaddons",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListInstallableAddonsV2Reader{formats: a.formats},
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
	success, ok := result.(*ListInstallableAddonsV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListInstallableAddonsV2Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  PatchAddon patches an addon that is assigned to the given cluster
*/
func (a *Client) PatchAddon(params *PatchAddonParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PatchAddonOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPatchAddonParams()
	}
	op := &runtime.ClientOperation{
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
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*PatchAddonOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PatchAddonDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  PatchAddonV2 patches an addon that is assigned to the given cluster
*/
func (a *Client) PatchAddonV2(params *PatchAddonV2Params, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*PatchAddonV2OK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewPatchAddonV2Params()
	}
	op := &runtime.ClientOperation{
		ID:                 "patchAddonV2",
		Method:             "PATCH",
		PathPattern:        "/api/v2/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &PatchAddonV2Reader{formats: a.formats},
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
	success, ok := result.(*PatchAddonV2OK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*PatchAddonV2Default)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
