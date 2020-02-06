// Code generated by go-swagger; DO NOT EDIT.

package users

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// EditUserInProjectReader is a Reader for the EditUserInProject structure.
type EditUserInProjectReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *EditUserInProjectReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewEditUserInProjectOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewEditUserInProjectUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewEditUserInProjectForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewEditUserInProjectDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewEditUserInProjectOK creates a EditUserInProjectOK with default headers values
func NewEditUserInProjectOK() *EditUserInProjectOK {
	return &EditUserInProjectOK{}
}

/*EditUserInProjectOK handles this case with default header values.

User
*/
type EditUserInProjectOK struct {
	Payload *models.User
}

func (o *EditUserInProjectOK) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/users/{user_id}][%d] editUserInProjectOK  %+v", 200, o.Payload)
}

func (o *EditUserInProjectOK) GetPayload() *models.User {
	return o.Payload
}

func (o *EditUserInProjectOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.User)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewEditUserInProjectUnauthorized creates a EditUserInProjectUnauthorized with default headers values
func NewEditUserInProjectUnauthorized() *EditUserInProjectUnauthorized {
	return &EditUserInProjectUnauthorized{}
}

/*EditUserInProjectUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type EditUserInProjectUnauthorized struct {
}

func (o *EditUserInProjectUnauthorized) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/users/{user_id}][%d] editUserInProjectUnauthorized ", 401)
}

func (o *EditUserInProjectUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewEditUserInProjectForbidden creates a EditUserInProjectForbidden with default headers values
func NewEditUserInProjectForbidden() *EditUserInProjectForbidden {
	return &EditUserInProjectForbidden{}
}

/*EditUserInProjectForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type EditUserInProjectForbidden struct {
}

func (o *EditUserInProjectForbidden) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/users/{user_id}][%d] editUserInProjectForbidden ", 403)
}

func (o *EditUserInProjectForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewEditUserInProjectDefault creates a EditUserInProjectDefault with default headers values
func NewEditUserInProjectDefault(code int) *EditUserInProjectDefault {
	return &EditUserInProjectDefault{
		_statusCode: code,
	}
}

/*EditUserInProjectDefault handles this case with default header values.

errorResponse
*/
type EditUserInProjectDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the edit user in project default response
func (o *EditUserInProjectDefault) Code() int {
	return o._statusCode
}

func (o *EditUserInProjectDefault) Error() string {
	return fmt.Sprintf("[PUT /api/v1/projects/{project_id}/users/{user_id}][%d] editUserInProject default  %+v", o._statusCode, o.Payload)
}

func (o *EditUserInProjectDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *EditUserInProjectDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
