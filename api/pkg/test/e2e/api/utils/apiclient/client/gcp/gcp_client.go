// Code generated by go-swagger; DO NOT EDIT.

package gcp

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"
)

// New creates a new gcp API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) *Client {
	return &Client{transport: transport, formats: formats}
}

/*
Client for gcp API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

/*
ListGCPDiskTypes Lists disk types from GCP
*/
func (a *Client) ListGCPDiskTypes(params *ListGCPDiskTypesParams, authInfo runtime.ClientAuthInfoWriter) (*ListGCPDiskTypesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListGCPDiskTypesParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listGCPDiskTypes",
		Method:             "GET",
		PathPattern:        "/api/v1/providers/gcp/disktypes",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListGCPDiskTypesReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListGCPDiskTypesOK), nil

}

/*
ListGCPDiskTypesNoCredentials Lists disk types from GCP
*/
func (a *Client) ListGCPDiskTypesNoCredentials(params *ListGCPDiskTypesNoCredentialsParams, authInfo runtime.ClientAuthInfoWriter) (*ListGCPDiskTypesNoCredentialsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListGCPDiskTypesNoCredentialsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listGCPDiskTypesNoCredentials",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/gcp/disktypes",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListGCPDiskTypesNoCredentialsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListGCPDiskTypesNoCredentialsOK), nil

}

/*
ListGCPSizes Lists machine sizes from GCP
*/
func (a *Client) ListGCPSizes(params *ListGCPSizesParams, authInfo runtime.ClientAuthInfoWriter) (*ListGCPSizesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListGCPSizesParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listGCPSizes",
		Method:             "GET",
		PathPattern:        "/api/v1/providers/gcp/sizes",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListGCPSizesReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListGCPSizesOK), nil

}

/*
ListGCPSizesNoCredentials Lists machine sizes from GCP
*/
func (a *Client) ListGCPSizesNoCredentials(params *ListGCPSizesNoCredentialsParams, authInfo runtime.ClientAuthInfoWriter) (*ListGCPSizesNoCredentialsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListGCPSizesNoCredentialsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listGCPSizesNoCredentials",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/gcp/sizes",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListGCPSizesNoCredentialsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListGCPSizesNoCredentialsOK), nil

}

/*
ListGCPZones Lists available GCP zones
*/
func (a *Client) ListGCPZones(params *ListGCPZonesParams, authInfo runtime.ClientAuthInfoWriter) (*ListGCPZonesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListGCPZonesParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listGCPZones",
		Method:             "GET",
		PathPattern:        "/api/v1/providers/gcp/{dc}/zones",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListGCPZonesReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListGCPZonesOK), nil

}

/*
ListGCPZonesNoCredentials Lists available GCP zones
*/
func (a *Client) ListGCPZonesNoCredentials(params *ListGCPZonesNoCredentialsParams, authInfo runtime.ClientAuthInfoWriter) (*ListGCPZonesNoCredentialsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListGCPZonesNoCredentialsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listGCPZonesNoCredentials",
		Method:             "GET",
		PathPattern:        "/api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/providers/gcp/zones",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &ListGCPZonesNoCredentialsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListGCPZonesNoCredentialsOK), nil

}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
