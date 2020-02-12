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

// DeleteUserFromProjectReader is a Reader for the DeleteUserFromProject structure.
type DeleteUserFromProjectReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteUserFromProjectReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteUserFromProjectOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeleteUserFromProjectUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewDeleteUserFromProjectForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewDeleteUserFromProjectDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteUserFromProjectOK creates a DeleteUserFromProjectOK with default headers values
func NewDeleteUserFromProjectOK() *DeleteUserFromProjectOK {
	return &DeleteUserFromProjectOK{}
}

/*DeleteUserFromProjectOK handles this case with default header values.

User
*/
type DeleteUserFromProjectOK struct {
	Payload *models.User
}

func (o *DeleteUserFromProjectOK) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/users/{user_id}][%d] deleteUserFromProjectOK  %+v", 200, o.Payload)
}

func (o *DeleteUserFromProjectOK) GetPayload() *models.User {
	return o.Payload
}

func (o *DeleteUserFromProjectOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.User)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewDeleteUserFromProjectUnauthorized creates a DeleteUserFromProjectUnauthorized with default headers values
func NewDeleteUserFromProjectUnauthorized() *DeleteUserFromProjectUnauthorized {
	return &DeleteUserFromProjectUnauthorized{}
}

/*DeleteUserFromProjectUnauthorized handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteUserFromProjectUnauthorized struct {
}

func (o *DeleteUserFromProjectUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/users/{user_id}][%d] deleteUserFromProjectUnauthorized ", 401)
}

func (o *DeleteUserFromProjectUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteUserFromProjectForbidden creates a DeleteUserFromProjectForbidden with default headers values
func NewDeleteUserFromProjectForbidden() *DeleteUserFromProjectForbidden {
	return &DeleteUserFromProjectForbidden{}
}

/*DeleteUserFromProjectForbidden handles this case with default header values.

EmptyResponse is a empty response
*/
type DeleteUserFromProjectForbidden struct {
}

func (o *DeleteUserFromProjectForbidden) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/users/{user_id}][%d] deleteUserFromProjectForbidden ", 403)
}

func (o *DeleteUserFromProjectForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeleteUserFromProjectDefault creates a DeleteUserFromProjectDefault with default headers values
func NewDeleteUserFromProjectDefault(code int) *DeleteUserFromProjectDefault {
	return &DeleteUserFromProjectDefault{
		_statusCode: code,
	}
}

/*DeleteUserFromProjectDefault handles this case with default header values.

errorResponse
*/
type DeleteUserFromProjectDefault struct {
	_statusCode int

	Payload *models.ErrorResponse
}

// Code gets the status code for the delete user from project default response
func (o *DeleteUserFromProjectDefault) Code() int {
	return o._statusCode
}

func (o *DeleteUserFromProjectDefault) Error() string {
	return fmt.Sprintf("[DELETE /api/v1/projects/{project_id}/users/{user_id}][%d] deleteUserFromProject default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteUserFromProjectDefault) GetPayload() *models.ErrorResponse {
	return o.Payload
}

func (o *DeleteUserFromProjectDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ErrorResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
