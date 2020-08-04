// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/models"
)

// ListNodesForClusterLegacyReader is a Reader for the ListNodesForClusterLegacy structure.
type ListNodesForClusterLegacyReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListNodesForClusterLegacyReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListNodesForClusterLegacyOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListNodesForClusterLegacyUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListNodesForClusterLegacyForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListNodesForClusterLegacyDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListNodesForClusterLegacyOK creates a ListNodesForClusterLegacyOK with default headers values
func NewListNodesForClusterLegacyOK() *ListNodesForClusterLegacyOK {
	return &ListNodesForClusterLegacyOK{}
}

/*ListNodesForClusterLegacyOK handles this case with default header values.

Node
*/
type ListNodesForClusterLegacyOK struct {
	Payload []*models.Node
}

func (o *ListNodesForClusterLegacyOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes][%d] listNodesForClusterLegacyOK  %+v", 200, o.Payload)
}

func (o *ListNodesForClusterLegacyOK) GetPayload() []*models.Node {
	return o.Payload
}

func (o *ListNodesForClusterLegacyOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListNodesForClusterLegacyUnauthorized creates a ListNodesForClusterLegacyUnauthorized with default headers values
func NewListNodesForClusterLegacyUnauthorized() *ListNodesForClusterLegacyUnauthorized {
	return &ListNodesForClusterLegacyUnauthorized{}
}

/*ListNodesForClusterLegacyUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type ListNodesForClusterLegacyUnauthorized struct {
}

func (o *ListNodesForClusterLegacyUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes][%d] listNodesForClusterLegacyUnauthorized ", 401)
}

func (o *ListNodesForClusterLegacyUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListNodesForClusterLegacyForbidden creates a ListNodesForClusterLegacyForbidden with default headers values
func NewListNodesForClusterLegacyForbidden() *ListNodesForClusterLegacyForbidden {
	return &ListNodesForClusterLegacyForbidden{}
}

/*ListNodesForClusterLegacyForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type ListNodesForClusterLegacyForbidden struct {
}

func (o *ListNodesForClusterLegacyForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes][%d] listNodesForClusterLegacyForbidden ", 403)
}

func (o *ListNodesForClusterLegacyForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListNodesForClusterLegacyDefault creates a ListNodesForClusterLegacyDefault with default headers values
func NewListNodesForClusterLegacyDefault(code int) *ListNodesForClusterLegacyDefault {
	return &ListNodesForClusterLegacyDefault{
		_statusCode: code,
	}
}

/*ListNodesForClusterLegacyDefault handles this case with default header values.

errorResponse
*/
type ListNodesForClusterLegacyDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list nodes for cluster legacy default response
func (o *ListNodesForClusterLegacyDefault) Code() int {
	return o._statusCode
}

func (o *ListNodesForClusterLegacyDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects/{project_id}/dc/{dc}/clusters/{cluster_id}/nodes][%d] listNodesForClusterLegacy default  %+v", o._statusCode, o.Payload)
}

func (o *ListNodesForClusterLegacyDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListNodesForClusterLegacyDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
