// Code generated by go-swagger; DO NOT EDIT.

package project

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

// ListClustersV2Reader is a Reader for the ListClustersV2 structure.
type ListClustersV2Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListClustersV2Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListClustersV2OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListClustersV2Unauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListClustersV2Forbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListClustersV2Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListClustersV2OK creates a ListClustersV2OK with default headers values
func NewListClustersV2OK() *ListClustersV2OK {
	return &ListClustersV2OK{}
}

/* ListClustersV2OK describes a response with status code 200, with default header values.

ClusterList
*/
type ListClustersV2OK struct {
	Payload models.ClusterList
}

func (o *ListClustersV2OK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters][%d] listClustersV2OK  %+v", 200, o.Payload)
}
func (o *ListClustersV2OK) GetPayload() models.ClusterList {
	return o.Payload
}

func (o *ListClustersV2OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListClustersV2Unauthorized creates a ListClustersV2Unauthorized with default headers values
func NewListClustersV2Unauthorized() *ListClustersV2Unauthorized {
	return &ListClustersV2Unauthorized{}
}

/* ListClustersV2Unauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListClustersV2Unauthorized struct {
}

func (o *ListClustersV2Unauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters][%d] listClustersV2Unauthorized ", 401)
}

func (o *ListClustersV2Unauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListClustersV2Forbidden creates a ListClustersV2Forbidden with default headers values
func NewListClustersV2Forbidden() *ListClustersV2Forbidden {
	return &ListClustersV2Forbidden{}
}

/* ListClustersV2Forbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListClustersV2Forbidden struct {
}

func (o *ListClustersV2Forbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters][%d] listClustersV2Forbidden ", 403)
}

func (o *ListClustersV2Forbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListClustersV2Default creates a ListClustersV2Default with default headers values
func NewListClustersV2Default(code int) *ListClustersV2Default {
	return &ListClustersV2Default{
		_statusCode: code,
	}
}

/* ListClustersV2Default describes a response with status code -1, with default header values.

errorResponse
*/
type ListClustersV2Default struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list clusters v2 default response
func (o *ListClustersV2Default) Code() int {
	return o._statusCode
}

func (o *ListClustersV2Default) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters][%d] listClustersV2 default  %+v", o._statusCode, o.Payload)
}
func (o *ListClustersV2Default) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListClustersV2Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
