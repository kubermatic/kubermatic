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

// CreateConstraintReader is a Reader for the CreateConstraint structure.
type CreateConstraintReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateConstraintReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewCreateConstraintOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewCreateConstraintUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewCreateConstraintForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewCreateConstraintDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateConstraintOK creates a CreateConstraintOK with default headers values
func NewCreateConstraintOK() *CreateConstraintOK {
	return &CreateConstraintOK{}
}

/*CreateConstraintOK handles this case with default header values.

Constraint
*/
type CreateConstraintOK struct {
	Payload *models.Constraint
}

func (o *CreateConstraintOK) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints][%d] createConstraintOK  %+v", 200, o.Payload)
}

func (o *CreateConstraintOK) GetPayload() *models.Constraint {
	return o.Payload
}

func (o *CreateConstraintOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Constraint)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateConstraintUnauthorized creates a CreateConstraintUnauthorized with default headers values
func NewCreateConstraintUnauthorized() *CreateConstraintUnauthorized {
	return &CreateConstraintUnauthorized{}
}

/*CreateConstraintUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateConstraintUnauthorized struct {
}

func (o *CreateConstraintUnauthorized) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints][%d] createConstraintUnauthorized ", 401)
}

func (o *CreateConstraintUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateConstraintForbidden creates a CreateConstraintForbidden with default headers values
func NewCreateConstraintForbidden() *CreateConstraintForbidden {
	return &CreateConstraintForbidden{}
}

/*CreateConstraintForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type CreateConstraintForbidden struct {
}

func (o *CreateConstraintForbidden) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints][%d] createConstraintForbidden ", 403)
}

func (o *CreateConstraintForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewCreateConstraintDefault creates a CreateConstraintDefault with default headers values
func NewCreateConstraintDefault(code int) *CreateConstraintDefault {
	return &CreateConstraintDefault{
		_statusCode: code,
	}
}

/*CreateConstraintDefault handles this case with default header values.

errorResponse
*/
type CreateConstraintDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the create constraint default response
func (o *CreateConstraintDefault) Code() int {
	return o._statusCode
}

func (o *CreateConstraintDefault) Error() string {
	return fmt.Sprintf("[POST /api/v2/projects/{project_id}/clusters/{cluster_id}/constraints][%d] createConstraint default  %+v", o._statusCode, o.Payload)
}

func (o *CreateConstraintDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *CreateConstraintDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
