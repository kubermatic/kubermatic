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

// ListProjectsReader is a Reader for the ListProjects structure.
type ListProjectsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListProjectsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewListProjectsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewListProjectsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 409:
		result := NewListProjectsConflict()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewListProjectsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListProjectsOK creates a ListProjectsOK with default headers values
func NewListProjectsOK() *ListProjectsOK {
	return &ListProjectsOK{}
}

/* ListProjectsOK describes a response with status code 200, with default header values.

Project
*/
type ListProjectsOK struct {
	Payload []*models.Project
}

func (o *ListProjectsOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects][%d] listProjectsOK  %+v", 200, o.Payload)
}
func (o *ListProjectsOK) GetPayload() []*models.Project {
	return o.Payload
}

func (o *ListProjectsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListProjectsUnauthorized creates a ListProjectsUnauthorized with default headers values
func NewListProjectsUnauthorized() *ListProjectsUnauthorized {
	return &ListProjectsUnauthorized{}
}

/* ListProjectsUnauthorized describes a response with status code 401, with default header values.

EmptyResponse is a empty response
*/
type ListProjectsUnauthorized struct {
}

func (o *ListProjectsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects][%d] listProjectsUnauthorized ", 401)
}

func (o *ListProjectsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListProjectsConflict creates a ListProjectsConflict with default headers values
func NewListProjectsConflict() *ListProjectsConflict {
	return &ListProjectsConflict{}
}

/* ListProjectsConflict describes a response with status code 409, with default header values.

EmptyResponse is a empty response
*/
type ListProjectsConflict struct {
}

func (o *ListProjectsConflict) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects][%d] listProjectsConflict ", 409)
}

func (o *ListProjectsConflict) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewListProjectsDefault creates a ListProjectsDefault with default headers values
func NewListProjectsDefault(code int) *ListProjectsDefault {
	return &ListProjectsDefault{
		_statusCode: code,
	}
}

/* ListProjectsDefault describes a response with status code -1, with default header values.

errorResponse
*/
type ListProjectsDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the list projects default response
func (o *ListProjectsDefault) Code() int {
	return o._statusCode
}

func (o *ListProjectsDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/projects][%d] listProjects default  %+v", o._statusCode, o.Payload)
}
func (o *ListProjectsDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *ListProjectsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
