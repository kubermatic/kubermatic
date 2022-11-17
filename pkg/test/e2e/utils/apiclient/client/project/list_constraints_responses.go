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

// ListConstraintsReader is a Reader for the ListConstraints structure.
type ListConstraintsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListConstraintsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListConstraintsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListConstraintsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewListConstraintsForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListConstraintsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListConstraintsOK creates a ListConstraintsOK with default headers values
func NewListConstraintsOK() *ListConstraintsOK {
	return &ListConstraintsOK{}
}

/* ListConstraintsOK describes a response with status code 200, with default header values.

Constraint
*/
type ListConstraintsOK struct {
	Payload []*models.Constraint
}

func (o *ListConstraintsOK) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints][%d] listConstraintsOK  %+v", 200, o.Payload)
}
func (o *ListConstraintsOK) GetPayload() []*models.Constraint {
	return o.Payload
}

func (o *ListConstraintsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListConstraintsUnauthorized creates a ListConstraintsUnauthorized with default headers values
func NewListConstraintsUnauthorized() *ListConstraintsUnauthorized {
	return &ListConstraintsUnauthorized{}
}

/* ListConstraintsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListConstraintsUnauthorized struct {
}

func (o *ListConstraintsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints][%d] listConstraintsUnauthorized ", 401)
}

func (o *ListConstraintsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListConstraintsForbidden creates a ListConstraintsForbidden with default headers values
func NewListConstraintsForbidden() *ListConstraintsForbidden {
	return &ListConstraintsForbidden{}
}

/* ListConstraintsForbidden describes a response with status code 403, with default header values.

EmptyResponse is a empty response
*/
type ListConstraintsForbidden struct {
}

func (o *ListConstraintsForbidden) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints][%d] listConstraintsForbidden ", 403)
}

func (o *ListConstraintsForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListConstraintsDefault creates a ListConstraintsDefault with default headers values
func NewListConstraintsDefault(code int) *ListConstraintsDefault {
	return &ListConstraintsDefault{
		_statusCode: code,
	}
}

/* ListConstraintsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListConstraintsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list constraints default response
func (o *ListConstraintsDefault) Code() int {
	return o._statusCode
}

func (o *ListConstraintsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints][%d] listConstraints default  %+v", o._statusCode, o.Payload)
}
func (o *ListConstraintsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListConstraintsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}