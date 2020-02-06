// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// GetNodeForClusterLegacyReader is a Reader for the GetNodeForClusterLegacy structure.
type GetNodeForClusterLegacyReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetNodeForClusterLegacyReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetNodeForClusterLegacyOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetNodeForClusterLegacyUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewGetNodeForClusterLegacyForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewGetNodeForClusterLegacyDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetNodeForClusterLegacyOK creates a GetNodeForClusterLegacyOK with default headers values
func NewGetNodeForClusterLegacyOK() *GetNodeForClusterLegacyOK {
	return &GetNodeForClusterLegacyOK{}
}

/*GetNodeForClusterLegacyOK handles this case with default header values.

Node
*/
type GetNodeForClusterLegacyOK struct {
	Payload *models.Node
}

func (o *GetNodeForClusterLegacyOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id}][%d] getNodeForClusterLegacyOK  %+v", 200, o.Payload)
}

func (o *GetNodeForClusterLegacyOK) GetPayload() *models.Node {
	return o.Payload
}

func (o *GetNodeForClusterLegacyOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Node)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetNodeForClusterLegacyUnauthorized creates a GetNodeForClusterLegacyUnauthorized with default headers values
func NewGetNodeForClusterLegacyUnauthorized() *GetNodeForClusterLegacyUnauthorized {
	return &GetNodeForClusterLegacyUnauthorized{}
}

/*GetNodeForClusterLegacyUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type GetNodeForClusterLegacyUnauthorized struct {
}

func (o *GetNodeForClusterLegacyUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id}][%d] getNodeForClusterLegacyUnauthorized ", 401)
}

func (o *GetNodeForClusterLegacyUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetNodeForClusterLegacyForbidden creates a GetNodeForClusterLegacyForbidden with default headers values
func NewGetNodeForClusterLegacyForbidden() *GetNodeForClusterLegacyForbidden {
	return &GetNodeForClusterLegacyForbidden{}
}

/*GetNodeForClusterLegacyForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type GetNodeForClusterLegacyForbidden struct {
}

func (o *GetNodeForClusterLegacyForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id}][%d] getNodeForClusterLegacyForbidden ", 403)
}

func (o *GetNodeForClusterLegacyForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewGetNodeForClusterLegacyDefault creates a GetNodeForClusterLegacyDefault with default headers values
func NewGetNodeForClusterLegacyDefault(code int) *GetNodeForClusterLegacyDefault {
	return &GetNodeForClusterLegacyDefault{
		_statusCode: code,
	}
}

/*GetNodeForClusterLegacyDefault handles this case with default header values.

errorResponse
*/
type GetNodeForClusterLegacyDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the get node for cluster legacy default response
func (o *GetNodeForClusterLegacyDefault) Code() int {
	return o._statusCode
}

func (o *GetNodeForClusterLegacyDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes/{node_id}][%d] getNodeForClusterLegacy default  %+v", o._statusCode, o.Payload)
}

func (o *GetNodeForClusterLegacyDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *GetNodeForClusterLegacyDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
