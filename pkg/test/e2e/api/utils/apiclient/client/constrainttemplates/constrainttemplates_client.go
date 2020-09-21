// Code generated by go-swagger; DO NOT EDIT.

package constrainttemplates

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new constrainttemplates API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for constrainttemplates API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientService is the interface for Client methods
type ClientService interface {
	CreateConstraintTemplate(params *CreateConstraintTemplateParams, authInfo runtime.ClientAuthInfoWriter) (*CreateConstraintTemplateOK, error)

	GetConstraintTemplate(params *GetConstraintTemplateParams, authInfo runtime.ClientAuthInfoWriter) (*GetConstraintTemplateOK, error)

	ListConstraintTemplates(params *ListConstraintTemplatesParams, authInfo runtime.ClientAuthInfoWriter) (*ListConstraintTemplatesOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  CreateConstraintTemplate Create constraint templates
*/
func (a *Client) CreateConstraintTemplate(params *CreateConstraintTemplateParams, authInfo runtime.ClientAuthInfoWriter) (*CreateConstraintTemplateOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewCreateConstraintTemplateParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "createConstraintTemplate",
		Method:             "POST",
		PathPattern:        "/api/v2/constrainttemplates",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &CreateConstraintTemplateReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*CreateConstraintTemplateOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*CreateConstraintTemplateDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  GetConstraintTemplate Get constraint templates specified by name
*/
func (a *Client) GetConstraintTemplate(params *GetConstraintTemplateParams, authInfo runtime.ClientAuthInfoWriter) (*GetConstraintTemplateOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetConstraintTemplateParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "getConstraintTemplate",
		Method:             "GET",
		PathPattern:        "/api/v2/constrainttemplates/{ct_name}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetConstraintTemplateReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetConstraintTemplateOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*GetConstraintTemplateDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

/*
  ListConstraintTemplates lists constraint templates
*/
func (a *Client) ListConstraintTemplates(params *ListConstraintTemplatesParams, authInfo runtime.ClientAuthInfoWriter) (*ListConstraintTemplatesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListConstraintTemplatesParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listConstraintTemplates",
		Method:             "GET",
		PathPattern:        "/api/v2/constrainttemplates",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ListConstraintTemplatesReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ListConstraintTemplatesOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	unexpectedSuccess := result.(*ListConstraintTemplatesDefault)
	return nil, runtime.NewAPIError("unexpected success response: content available as default response in error", unexpectedSuccess, unexpectedSuccess.Code())
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
